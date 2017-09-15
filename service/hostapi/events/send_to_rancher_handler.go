package events

import (
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/distribution/context"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"github.com/patrickmn/go-cache"
	"github.com/rancher/agent/utils"
	"github.com/rancher/event-subscriber/locks"
	rclient "github.com/rancher/go-rancher/v3"
)

type SendToRancherHandler struct {
	client   *client.Client
	rancher  *rclient.RancherClient
	hostUUID string
	cache    *cache.Cache
}

func (h *SendToRancherHandler) Handle(event *events.Message) error {
	// rancher_state_watcher sends a simulated event to the event router to initiate ip injection.
	// This event should not be sent.
	if event.From == simulatedEvent {
		return nil
	}

	// Note: event.ID == container's ID
	lock := locks.Lock(event.Status + event.ID)
	if lock == nil {
		log.Debugf("Container locked. Can't run SendToRancherHandler. Event: [%s], ID: [%s]", event.Status, event.ID)
		return nil
	}
	defer lock.Unlock()

	container, err := h.client.ContainerInspect(context.Background(), event.ID)
	if err != nil {
		if ok := client.IsErrContainerNotFound(err); !ok {
			return err
		}
	}

	if err == nil && strings.HasPrefix(container.Image, "sha256:") {
		if err := utils.ReplaceFriendlyImage(h.cache, h.client, &container); err != nil {
			return err
		}
	}

	if container.Config == nil {
		return nil
	}

	ok, uuid := utils.GetUUIDForContainer(container.Config.Labels)
	if !ok {
		return nil
	}

	containerEvent := &rclient.ContainerEvent{
		ExternalStatus:   event.Status,
		ExternalId:       event.ID,
		ContainerUuid:    uuid,
		ReportedHostUuid: h.hostUUID,
	}
	containerEvent.DockerInspect = container

	if _, err := h.rancher.ContainerEvent.Create(containerEvent); err != nil {
		return err
	}

	return nil
}
