package stats

import (
	"bufio"
	"io"
	"net/url"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/rancher/agent/service/hostapi/config"
	"github.com/rancher/websocket-proxy/backend"
	"github.com/rancher/websocket-proxy/common"
)

type HostStatsHandler struct {
}

func (s *HostStatsHandler) Handle(key string, initialMessage string, incomingMessages <-chan string, response chan<- common.Message) {
	defer backend.SignalHandlerClosed(key, response)

	requestURL, err := url.Parse(initialMessage)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "message": initialMessage}).Error("Couldn't parse url from message.")
		return
	}

	tokenString := requestURL.Query().Get("token")

	resourceID := ""

	token, err := parseRequestToken(tokenString, config.Config.ParsedPublicKey)
	if err == nil {
		resourceIDInterface, found := token.Claims["resourceId"]
		if found {
			resourceIDVal, ok := resourceIDInterface.(string)
			if ok {
				resourceID = resourceIDVal
			}
		}
	}

	reader, writer := io.Pipe()

	go func(w *io.PipeWriter) {
		for {
			_, ok := <-incomingMessages
			if !ok {
				w.Close()
				return
			}
		}
	}(writer)

	go func(r *io.PipeReader) {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			text := scanner.Text()
			message := common.Message{
				Key:  key,
				Type: common.Body,
				Body: text,
			}
			response <- message
		}
		if err := scanner.Err(); err != nil {
			log.WithFields(log.Fields{"error": err}).Error("Error with the container stat scanner.")
		}
	}(reader)

	memLimit, err := getMemCapcity()
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error getting memory capacity.")
		return
	}

	for {
		infos := []containerInfo{}

		cInfo, err := getRootContainerInfo()
		if err != nil {
			return
		}

		infos = append(infos, cInfo)
		for i := range infos {
			if len(infos[i].Stats) > 0 {
				infos[i].Stats[0].Timestamp = time.Now()
			}
		}

		err = writeAggregatedStats(resourceID, nil, "host", infos, uint64(memLimit), writer)
		if err != nil {
			return
		}

		time.Sleep(1 * time.Second)
	}
}
