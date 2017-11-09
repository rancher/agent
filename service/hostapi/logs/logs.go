package logs

import (
	"bufio"
	"bytes"
	"io"
	"net/url"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"

	"github.com/rancher/websocket-proxy/backend"
	"github.com/rancher/websocket-proxy/common"

	"github.com/docker/docker/api/types"
	"github.com/rancher/agent/service/hostapi/auth"
	"github.com/rancher/agent/service/hostapi/events"
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
		log.Error(err)
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
				log.WithFields(log.Fields{"error": err}).Error("Error with the container log scanner.")
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
