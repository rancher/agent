package logs

import (
	"bufio"
	"io"
	"net/url"
	"strconv"

	log "github.com/Sirupsen/logrus"

	"github.com/rancher/websocket-proxy/backend"
	"github.com/rancher/websocket-proxy/common"

	"bytes"
	"github.com/docker/distribution/context"
	"github.com/docker/docker/api/types"
	"github.com/rancher/agent/service/hostapi/auth"
	"github.com/rancher/agent/service/hostapi/events"
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
		log.WithFields(log.Fields{"error": err, "url": initialMessage}).Error("Couldn't parse url.")
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
		log.WithFields(log.Fields{"error": err}).Error("Couldn't get docker client.")
		return
	}

	bothPrefix := "00 "
	stdoutPrefix := "01 "
	stderrPrefix := "02 "
	logOpts := types.ContainerLogsOptions{
		Follow:     follow,
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Tail:       tail,
	}

	stdoutReader, err := client.ContainerLogs(context.Background(), container, logOpts)
	if err != nil {
		log.Error(err)
		return
	}

	go func() {
		for {
			_, ok := <-incomingMessages
			if !ok {
				return
			}
		}
	}()

	go func(stdout io.ReadCloser) {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			body := ""
			data := scanner.Bytes()
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
		if err := scanner.Err(); err != nil {
			log.WithFields(log.Fields{"error": err}).Error("Error with the container log scanner.")
		}
		stdout.Close()
	}(stdoutReader)

	select {}
}
