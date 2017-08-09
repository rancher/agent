package handlers

import (
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/rancher/agent/runtime"
	"github.com/rancher/agent/utils"
	revents "github.com/rancher/event-subscriber/events"
	v3 "github.com/rancher/go-rancher/v3"
)

func (h *StorageHandler) VolumeRemove(event *revents.Event, client *v3.RancherClient) error {
	volume := v3.Volume{}
	err := utils.Unmarshalling(event.Data["volume"], &volume)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshall volume")
	}
	progress := utils.GetProgress(event, client)

	if runtime.IsRancherVolume(volume) {
		err := runtime.VolumeRemoveFlex(volume)
		if err != nil {
			return err
		}
	} else {
		err := runtime.VolumeRemoveDocker(volume, h.dockerClient, progress)
		if err != nil {
			return err
		}
	}
	logrus.Infof("rancher id [%v]: Volume [%v] has been removed", event.ResourceID, volume.Name)
	return reply(nil, event, client)
}
