package utils

import (
	"github.com/nu7hatch/gouuid"
	revents "github.com/rancher/go-machine-service/events"
)

func ReplyData(event *revents.Event) *revents.Event {
	var result revents.Event
	if event.ReplyTo != "" {
		value, _ := uuid.NewV4()
		result = revents.Event{
			ID:            value.String(),
			Name:          event.ReplyTo,
			Data:          map[string]interface{}{},
			ResourceType:  event.ResourceType,
			ResourceID:    event.ResourceID,
			PreviousIds:   event.ID,
			PreviousNames: event.Name,
		}
	}
	return &result
}

func DoPingAction(event, resp *revents.Event) {
	if !DockerEnable() {
		return
	}
	addResource(event, resp)
	addInstance(event, resp)
}

func addResource(ping, pong *revents.Event) {
	if !pingIncludeResource(ping) {
		return
	}
	return

}
func addInstance(ping, pong *revents.Event) {
	return
}

func pingIncludeResource(ping *revents.Event) bool {
	_, ok := GetFieldsIfExist(ping.Data, "options", "resources")
	return ok
}
