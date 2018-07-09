package stats

import (
	"bufio"
	"io"
	"net/url"
	"time"

	"github.com/rancher/agent/service/hostapi/events"
	"github.com/rancher/log"
	"github.com/rancher/websocket-proxy/backend"
	"github.com/rancher/websocket-proxy/common"
	"golang.org/x/net/context"
)

type Handler struct {
}

func (s *Handler) Handle(key string, initialMessage string, incomingMessages <-chan string, response chan<- common.Message) {
	defer backend.SignalHandlerClosed(key, response)

	requestURL, err := url.Parse(initialMessage)
	if err != nil {
		log.Errorf("Couldn't parse url from message. url=%v error=%v", initialMessage, err)
		return
	}

	id := ""
	parts := pathParts(requestURL.Path)
	if len(parts) == 3 {
		id = parts[2]
	}

	dclient, err := events.NewDockerClient()
	if err != nil {
		log.Errorf("Can not get docker client. err: %v", err)
		return
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
			log.Errorf("Error with the container stat scanner. error=%v", err)
		}
	}(reader)

	memLimit, err := getMemCapcity()
	if err != nil {
		log.Errorf("Error getting memory capacity. error=%v", err)
		return
	}
	if id == "" {
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

			err = writeAggregatedStats("", nil, "host", infos, uint64(memLimit), writer)
			if err != nil {
				return
			}

			time.Sleep(1 * time.Second)
		}
	} else {
		inspect, err := dclient.ContainerInspect(context.Background(), id)
		if err != nil {
			log.Errorf("Can not inspect containers error=%v", err)
			return
		}
		statsReader, err := dclient.ContainerStats(context.Background(), id, true)
		if err != nil {
			log.Errorf("Can not get stats reader from docker error=%v", err)
			return
		}
		defer statsReader.Body.Close()
		pid := inspect.State.Pid
		for {
			infos := []containerInfo{}
			cInfo, err := getContainerStats(statsReader.Body, id, pid)

			if err != nil {
				log.Errorf("Error getting container info. id=%v error=%v", id, err)
				return
			}
			infos = append(infos, cInfo)
			for i := range infos {
				if len(infos[i].Stats) > 0 {
					infos[i].Stats[0].Timestamp = time.Now()
				}
			}

			err = writeAggregatedStats(id, nil, "container", infos, uint64(memLimit), writer)
			if err != nil {
				return
			}

			time.Sleep(1 * time.Second)
		}
	}
}
