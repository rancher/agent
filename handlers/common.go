package handlers

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/types"
	goUUID "github.com/nu7hatch/gouuid"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/hostInfo"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/docker"
	"github.com/rancher/agent/utilities/utils"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v2"
	"golang.org/x/net/context"
	"os"
	"time"
)

type Handler struct {
	compute      *ComputeHandler
	storage      *StorageHandler
	configUpdate *ConfigUpdateHandler
	ping         *PingHandler
	delegate     *DelegateRequestHandler
}

func GetHandlers() map[string]revents.EventHandler {
	handler := initializeHandlers()
	return map[string]revents.EventHandler{
		"compute.instance.activate":   logRequest(handler.compute.InstanceActivate),
		"compute.instance.deactivate": logRequest(handler.compute.InstanceDeactivate),
		"compute.instance.force.stop": logRequest(handler.compute.InstanceForceStop),
		"compute.instance.inspect":    logRequest(handler.compute.InstanceInspect),
		"compute.instance.pull":       logRequest(handler.compute.InstancePull),
		"compute.instance.remove":     logRequest(handler.compute.InstanceRemove),
		"storage.image.activate":      logRequest(handler.storage.ImageActivate),
		"storage.volume.activate":     logRequest(handler.storage.VolumeActivate),
		"storage.volume.deactivate":   logRequest(handler.storage.VolumeDeactivate),
		"storage.volume.remove":       logRequest(handler.storage.VolumeRemove),
		"delegate.request":            logRequest(handler.delegate.DelegateRequest),
		"ping":                        handler.ping.Ping,
		"config.update":               logRequest(handler.configUpdate.ConfigUpdate),
	}
}

func logRequest(f revents.EventHandler) revents.EventHandler {
	return func(event *revents.Event, cli *client.RancherClient) error {
		logrus.Infof("Received event: Name: %s, Event Id: %s, Resource Id: %s", event.Name, event.ID, event.ResourceID)
		return f(event, cli)
	}
}

func reply(replyData map[string]interface{}, event *revents.Event, cli *client.RancherClient) error {
	if replyData == nil {
		replyData = make(map[string]interface{})
	}
	uuid, err := getUUID()
	if err != nil {
		return errors.Wrap(err, "can not aasign uuid to reply event")
	}
	reply := &client.Publish{
		ResourceId:   event.ResourceID,
		PreviousIds:  []string{event.ID},
		ResourceType: event.ResourceType,
		Name:         event.ReplyTo,
		Data:         replyData,
		Time:         time.Now().UnixNano() / int64(time.Millisecond),
		Resource:     client.Resource{Id: uuid},
	}

	if reply.ResourceType != "agent" {
		logrus.Infof("Reply: %v, %v, %v:%v", event.ID, event.Name, reply.ResourceId, reply.ResourceType)
	}
	logrus.Debugf("Reply: %+v", reply)

	err = publishReply(reply, cli)
	if err != nil {
		return fmt.Errorf("Error sending reply %v: %v", event.ID, err)
	}
	return nil
}

func initializeHandlers() *Handler {
	client := docker.GetClient(constants.DefaultVersion)
	info := types.Info{}
	version := types.Version{}
	systemImages := map[string]string{}
	flags := [3]bool{}
	// initialize the info and version so we don't have to call docker API every time a ping request comes
	for i := 0; i < 10; i++ {
		in, err := client.Info(context.Background())
		if err == nil {
			info = in
			flags[0] = true
			break
		}
		time.Sleep(time.Duration(1) * time.Second)
	}
	for i := 0; i < 10; i++ {
		v, err := client.ServerVersion(context.Background())
		if err == nil {
			version = v
			flags[1] = true
			break
		}
		time.Sleep(time.Duration(1) * time.Second)
	}
	for i := 0; i < 10; i++ {
		ret, err := utils.GetAgentImage(client)
		if err == nil {
			systemImages = ret
			flags[2] = true
			break
		}
		time.Sleep(time.Duration(1) * time.Second)
	}
	// if we can't get the initialization data the program should exit
	if !flags[0] || !flags[1] || !flags[2] {
		logrus.Fatalf("Failed to initialize handlers. Exiting go-agent")
		os.Exit(1)
	}
	Collectors := []hostInfo.Collector{
		hostInfo.CPUCollector{},
		hostInfo.DiskCollector{
			Unit: 1048576,
			InfoData: model.InfoData{
				Info:    info,
				Version: version,
			},
		},
		hostInfo.IopsCollector{},
		hostInfo.MemoryCollector{
			Unit: 1024.00,
		},
		hostInfo.OSCollector{
			InfoData: model.InfoData{
				Info:    info,
				Version: version,
			},
		},
	}
	computerHandler := ComputeHandler{
		dockerClient: client,
		infoData: model.InfoData{
			Info:    info,
			Version: version,
		},
	}
	storageHandler := StorageHandler{
		dockerClient: client,
	}
	delegateHandler := DelegateRequestHandler{
		dockerClient: client,
	}
	pingHandler := PingHandler{
		dockerClient: client,
		collectors:   Collectors,
		SystemImage:  systemImages,
	}
	configHandler := ConfigUpdateHandler{}
	handler := Handler{
		compute:      &computerHandler,
		storage:      &storageHandler,
		ping:         &pingHandler,
		configUpdate: &configHandler,
		delegate:     &delegateHandler,
	}
	return &handler
}

func replyWithParent(replyData map[string]interface{}, event *revents.Event, parent *revents.Event, cli *client.RancherClient) error {
	childUUID, err := getUUID()
	if err != nil {
		return errors.Wrap(err, "can not aasign uuid to reply event")
	}
	child := map[string]interface{}{
		"resourceId":    event.ResourceID,
		"previousIds":   []string{event.ID},
		"resourceType":  event.ResourceType,
		"name":          event.ReplyTo,
		"data":          replyData,
		"id":            childUUID,
		"time":          time.Now().UnixNano() / int64(time.Millisecond),
		"previousNames": []string{event.Name},
	}
	parentUUID, err := getUUID()
	if err != nil {
		return errors.Wrap(err, "can not aasign uuid to reply event")
	}
	reply := &client.Publish{
		ResourceId:   parent.ResourceID,
		PreviousIds:  []string{parent.ID},
		ResourceType: parent.ResourceType,
		Name:         parent.ReplyTo,
		Data:         child,
		Time:         time.Now().UnixNano() / int64(time.Millisecond),
		Resource:     client.Resource{Id: parentUUID},
	}
	if parent.ReplyTo == "" {
		return nil
	}

	if reply.ResourceType != "agent" {
		logrus.Infof("Reply: %v, %v, %v:%v", event.ID, event.Name, reply.ResourceId, reply.ResourceType)
	}
	logrus.Debugf("Reply: %+v", reply)

	err = publishReply(reply, cli)
	if err != nil {
		return fmt.Errorf("Error sending reply %v: %v", event.ID, err)
	}
	return nil
}

func getUUID() (string, error) {
	newUUID, err := goUUID.NewV4()
	if err != nil {
		return "", errors.Wrap(err, "can't generate uuid")
	}
	return newUUID.String(), nil

}

func publishReply(reply *client.Publish, apiClient *client.RancherClient) error {
	_, err := apiClient.Publish.Create(reply)
	if err != nil {
		logrus.Error(err)
	}
	return err
}
