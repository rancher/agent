package logs

import (
	"bufio"
	"bytes"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/leodotcloud/log"
	"github.com/rancher/agent/service/hostapi/auth"
	"github.com/rancher/agent/service/hostapi/events"
	"github.com/rancher/websocket-proxy/backend"
	"github.com/rancher/websocket-proxy/common"
	"golang.org/x/net/context"
)

var (
	stdoutHead = []byte{1, 0, 0, 0}
	stderrHead = []byte{2, 0, 0, 0}
)

type Handler struct {
}

func (l *Handler) Handle(key string, initialMessage string, incomingMessages <-chan string, response chan<- common.Message) {
	defer backend.SignalHandlerClosed(key, response)

	requestURL, err := url.Parse(initialMessage)
	if err != nil {
		log.Errorf("Couldn't parse url. url=%v error=%v", initialMessage, err)
		return
	}
	tokenString := requestURL.Query().Get("token")
	token, valid := auth.GetAndCheckToken(tokenString)
	if !valid {
		return
	}

	logs := token.Claims["logs"].(map[string]interface{})
	container := logs["Container"].(string)
	follow, found := logs["Follow"].(bool)

	if !found {
		follow = true
	}

	tailTemp, found := logs["Lines"].(int)
	var tail string
	if found {
		tail = strconv.Itoa(int(tailTemp))
	} else {
		tail = "100"
	}

	client, err := events.NewDockerClient()
	if err != nil {
		log.Errorf("Couldn't get docker client. error=%v", err)
		return
	}

	logOpts := types.ContainerLogsOptions{
		Follow:     follow,
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Tail:       tail,
	}

	ctx, cancelFnc := context.WithCancel(context.Background())
	stdout, err := client.ContainerLogs(ctx, container, logOpts)
	if err != nil {
		log.Errorf("error fetching container logs: %v", err)
		return
	}
	defer stdout.Close()

	go func() {
		for {
			_, ok := <-incomingMessages
			if !ok {
				cancelFnc()
				return
			}
		}
	}()

	reader := bufio.NewReader(stdout)
	for {
		data, err := reader.ReadBytes('\n')
		if err != nil {
			// hacky, but can't do a type assertion on the cancellation error, which is the "normal" error received
			// when the logs are closed properly
			if err != io.EOF && !strings.Contains(err.Error(), "context canceled") {
				log.Errorf("Error with the container log scanner. error=%v", err)
			}
			break
		}
		processData(data, key, response)
	}
}

func processData(data []byte, key string, response chan<- common.Message) {
	body := ""
	bothPrefix := "00 "
	stdoutPrefix := "01 "
	stderrPrefix := "02 "
	if bytes.Contains(data, stdoutHead) {
		if len(data) > 8 {
			body = stdoutPrefix + string(data[8:])
		}
	} else if bytes.Contains(data, stderrHead) {
		if len(data) > 8 {
			body = stderrPrefix + string(data[8:])
		}
	} else {
		body = bothPrefix + string(data)
	}
	message := common.Message{
		Key:  key,
		Type: common.Body,
		Body: body,
	}
	response <- message
}
