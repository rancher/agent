package stats

import (
	"bufio"
	"io"
	"net/url"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/rancher/agent/service/hostapi/config"
	"github.com/rancher/agent/service/hostapi/events"
	"github.com/rancher/websocket-proxy/backend"
	"github.com/rancher/websocket-proxy/common"
	"golang.org/x/net/context"
)

type ContainerStatsHandler struct {
}

func (s *ContainerStatsHandler) Handle(key string, initialMessage string, incomingMessages <-chan string, response chan<- common.Message) {
	defer backend.SignalHandlerClosed(key, response)

	requestURL, err := url.Parse(initialMessage)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "message": initialMessage}).Error("Couldn't parse url from message.")
		return
	}

	tokenString := requestURL.Query().Get("token")

	containerIds := map[string]string{}

	token, err := parseRequestToken(tokenString, config.Config.ParsedPublicKey)
	if err == nil {
		containerIdsInterface, found := token.Claims["containerIds"]
		if found {
			containerIdsVal, ok := containerIdsInterface.(map[string]interface{})
			if ok {
				for key, val := range containerIdsVal {
					if containerIdsValString, ok := val.(string); ok {
						containerIds[key] = containerIdsValString
					}
				}
			}
		}
	}

	id := ""
	parts := pathParts(requestURL.Path)
	if len(parts) == 3 {
		id = parts[2]
	}

	if err != nil {
		log.WithFields(log.Fields{"id": id, "error": err}).Error("Couldn't find container for id.")
		return
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
			log.WithFields(log.Fields{"error": err}).Error("Error with the container stat scanner.")
		}
	}(reader)
	memLimit, err := getMemCapcity()
	if err != nil {
		log.WithFields(log.Fields{"error": err, "id": id}).Error("Error getting memory capacity.")
		return
	}
	// get single container stats
	if id != "" {
		inspect, err := dclient.ContainerInspect(context.Background(), id)
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Error("Can not inspect containers")
			return
		}
		pid := inspect.State.Pid
		statsReader, err := dclient.ContainerStats(context.Background(), id, true)
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Error("Can not get stats reader from docker")
			return
		}
		defer statsReader.Body.Close()
		for {
			infos := []containerInfo{}
			cInfo, err := getContainerStats(statsReader.Body, id, pid)
			if err != nil {
				log.WithFields(log.Fields{"error": err, "id": id}).Error("Error getting container info.")
				return
			}
			infos = append(infos, cInfo)
			for i := range infos {
				if len(infos[i].Stats) > 0 {
					infos[i].Stats[0].Timestamp = time.Now()
				}
			}

			err = writeAggregatedStats(id, containerIds, "container", infos, uint64(memLimit), writer)
			if err != nil {
				return
			}
		}
	} else {
		contList, err := dclient.ContainerList(context.Background(), types.ContainerListOptions{})
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Error("Can not list containers")
			return
		}
		readerMap := map[string]io.ReadCloser{}
		pidMap := map[string]int{}
		for _, cont := range contList {
			if _, ok := containerIds[cont.ID]; ok {
				inspect, err := dclient.ContainerInspect(context.Background(), cont.ID)
				if err != nil {
					log.WithFields(log.Fields{"error": err}).Error("Can not inspect containers")
					return
				}
				statsReader, err := dclient.ContainerStats(context.Background(), cont.ID, true)
				if err != nil {
					log.WithFields(log.Fields{"error": err}).Error("Can not get stats reader from docker")
					return
				}
				defer statsReader.Body.Close()
				pid := inspect.State.Pid
				readerMap[cont.ID] = statsReader.Body
				pidMap[cont.ID] = pid
			}
		}
		for {
			infos := []containerInfo{}
			for id, r := range readerMap {
				cInfo, err := getContainerStats(r, id, pidMap[id])
				if err != nil {
					log.WithFields(log.Fields{"error": err, "id": id}).Error("Error getting container info.")
					return
				}
				infos = append(infos, cInfo)
			}
			err = writeAggregatedStats(id, containerIds, "container", infos, uint64(memLimit), writer)
			if err != nil {
				return
			}
		}
	}
}
