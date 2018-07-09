package exec

import (
	"encoding/base64"
	"io"
	"net/url"
	"runtime"
	"strconv"
	"strings"

	"github.com/docker/distribution/context"
	"github.com/docker/docker/api/types"
	"github.com/rancher/agent/service/hostapi/auth"
	"github.com/rancher/agent/service/hostapi/events"
	"github.com/rancher/log"
	"github.com/rancher/websocket-proxy/backend"
	"github.com/rancher/websocket-proxy/common"
)

type Handler struct {
}

func (h *Handler) Handle(key string, initialMessage string, incomingMessages <-chan string, response chan<- common.Message) {
	defer backend.SignalHandlerClosed(key, response)

	requestURL, err := url.Parse(initialMessage)
	if err != nil {
		log.Errorf("Couldn't parse url=%v error=%v", initialMessage, err)
		return
	}
	tokenString := requestURL.Query().Get("token")
	token, valid := auth.GetAndCheckToken(tokenString)
	if !valid {
		return
	}

	execMap := token.Claims["exec"].(map[string]interface{})
	execConfig, id := convert(execMap)

	client, err := events.NewDockerClient()
	if err != nil {
		log.Errorf("Couldn't get docker client. error=%v", err)
		return
	}

	execObj, err := client.ContainerExecCreate(context.Background(), id, execConfig)
	if err != nil {
		return
	}
	hijackResp, err := client.ContainerExecAttach(context.Background(), execObj.ID, execConfig)
	if err != nil {
		return
	}

	go func(w io.WriteCloser) {
		for {
			msg, ok := <-incomingMessages
			if !ok {
				if _, err := w.Write([]byte("\x04")); err != nil {
					log.Errorf("Error writing EOT message. error=%v", err)
				}
				w.Close()
				return
			}
			if strings.HasPrefix(msg, ":resizeTTY:") {
				resizeCommand := strings.Split(strings.Split(msg, ":")[2], ",")
				if len(resizeCommand) != 2 {
					continue
				}
				var width uint64
				width, err = strconv.ParseUint(resizeCommand[0], 10, 64)
				if err != nil {
					log.Errorf("Error decoding TTY width. error=%v", err)
					continue
				}
				var height uint64
				height, err = strconv.ParseUint(resizeCommand[1], 10, 64)
				if err != nil {
					log.Errorf("Error decoding TTY height. error=%v", err)
					continue
				}
				resizeOptions := types.ResizeOptions{}
				resizeOptions.Width = uint(width)
				resizeOptions.Height = uint(height)
				client.ContainerExecResize(context.Background(), execObj.ID, resizeOptions)
				continue
			}
			data, err := base64.StdEncoding.DecodeString(msg)
			if err != nil {
				log.Errorf("Error decoding message. error=%v", err)
				continue
			}
			w.Write([]byte(data))
		}
	}(hijackResp.Conn)

	buffer := make([]byte, 4096, 4096)
	for {
		c, err := hijackResp.Reader.Read(buffer)
		if c > 0 {
			text := base64.StdEncoding.EncodeToString(buffer[:c])
			message := common.Message{
				Key:  key,
				Type: common.Body,
				Body: text,
			}
			response <- message
		}
		if err != nil {
			break
		}
	}
}

func convert(execMap map[string]interface{}) (types.ExecConfig, string) {
	// Not fancy at all
	config := types.ExecConfig{}
	containerID := ""

	if param, ok := execMap["AttachStdin"]; ok {
		if val, ok := param.(bool); ok {
			config.AttachStdin = val
		}
	}

	if param, ok := execMap["AttachStdout"]; ok {
		if val, ok := param.(bool); ok {
			config.AttachStdout = val
		}
	}

	if param, ok := execMap["AttachStderr"]; ok {
		if val, ok := param.(bool); ok {
			config.AttachStderr = val
		}
	}

	if param, ok := execMap["Tty"]; ok {
		if val, ok := param.(bool); ok {
			config.Tty = val
		}
	}

	if param, ok := execMap["Container"]; ok {
		if val, ok := param.(string); ok {
			containerID = val
		}
	}

	if param, ok := execMap["Cmd"]; ok {
		cmd := []string{}
		if list, ok := param.([]interface{}); ok {
			for _, item := range list {
				if val, ok := item.(string); ok {
					cmd = append(cmd, val)
				}
			}
		}
		config.Cmd = cmd
	}

	if runtime.GOOS == "windows" {
		config.Cmd = []string{"powershell"}
	}

	return config, containerID
}
