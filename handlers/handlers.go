package handlers

import (
	"fmt"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	goUUID "github.com/nu7hatch/gouuid"
	"github.com/pkg/errors"
	"github.com/rancher/agent/host_info"
	"github.com/rancher/agent/utils"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v2"
	"golang.org/x/net/context"
)

func GetHandlers() (map[string]revents.EventHandler, error) {
	handler := initializeHandlers()

	utils.SerializeCompute = hostInfo.DockerData.Info.Driver == "devicemapper"
	logrus.Infof("Serialize compute requests: %v, driver: %s", utils.SerializeCompute, hostInfo.DockerData.Info.Driver)

	return map[string]revents.EventHandler{
		"compute.instance.activate":   cleanLog(logRequest(handler.compute.InstanceActivate)),
		"compute.instance.deactivate": utils.SerializeHandler(cleanLog(logRequest(handler.compute.InstanceDeactivate))),
		"compute.instance.inspect":    cleanLog(logRequest(handler.compute.InstanceInspect)),
		"compute.instance.pull":       cleanLog(logRequest(handler.compute.InstancePull)),
		"compute.instance.remove":     utils.SerializeHandler(cleanLog(logRequest(handler.compute.InstanceRemove))),
		"storage.volume.remove":       cleanLog(logRequest(handler.storage.VolumeRemove)),
		"ping":                        cleanLog(handler.ping.Ping),
	}, nil
}

func logRequest(f revents.EventHandler) revents.EventHandler {
	return func(event *revents.Event, cli *client.RancherClient) error {
		logrus.Infof("Received event: Name: %s, Event Id: %s, Resource Id: %s", event.Name, event.ID, event.ResourceID)
		startTime := time.Now()
		err := f(event, cli)
		logrus.Infof("Name: %s, Event Id: %s, Resource Id: %s, Process duration: %.4f seconds", event.Name, event.ID, event.ResourceID, time.Now().Sub(startTime).Seconds())
		return err
	}
}

func cleanLog(f revents.EventHandler) revents.EventHandler {
	return func(event *revents.Event, cli *client.RancherClient) error {
		err := f(event, cli)
		if err != nil {
			logrus.WithFields(logrus.Fields{"err": err}).Info("Verbose error message")
		}
		return errors.Cause(err)
	}
}

func reply(replyData map[string]interface{}, event *revents.Event, cli *client.RancherClient) error {
	if replyData == nil {
		replyData = make(map[string]interface{})
	}
	uuid, err := getUUID()
	if err != nil {
		return errors.Wrap(err, "can not assign uuid to reply event")
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
	runtime := utils.DefaultValue("RUNTIME", "docker")
	ver := utils.DefaultValue("RUNTIME_VERSION", "v1.22")
	runtimeClient := utils.GetRuntimeClient(runtime, ver)

	// initialize the info and version so we don't have to call docker API every time a ping request comes
	info := types.Info{}
	version := types.Version{}
	flags := [2]bool{}
	for i := 0; i < 10; i++ {
		in, err := runtimeClient.Info(context.Background())
		if err == nil {
			info = in
			flags[0] = true
			break
		}
		logrus.Error(err)
		time.Sleep(time.Duration(1) * time.Second)
	}
	for i := 0; i < 10; i++ {
		v, err := runtimeClient.ServerVersion(context.Background())
		if err == nil {
			version = v
			flags[1] = true
			break
		}
		logrus.Error(err)
		time.Sleep(time.Duration(1) * time.Second)
	}
	// if we can't get the initialization data the program should exit
	if !flags[0] || !flags[1] {
		logrus.Fatalf("Failed to initialize handlers. Exiting go-agent")
		os.Exit(1)
	}
	hostInfo.DockerData.Info = info
	hostInfo.DockerData.Version = version

	Collectors := []hostInfo.Collector{
		hostInfo.CPUCollector{},
		hostInfo.DiskCollector{
			Unit: 1048576,
		},
		hostInfo.IopsCollector{},
		hostInfo.MemoryCollector{
			Unit: 1024.00,
		},
		hostInfo.OSCollector{},
		hostInfo.KeyCollector{},
		hostInfo.CloudProviderCollector{},
	}
	computerHandler := ComputeHandler{
		dockerClient: runtimeClient,
	}
	storageHandler := StorageHandler{
		dockerClient: runtimeClient,
	}
	pingHandler := PingHandler{
		dockerClient: runtimeClient,
		collectors:   Collectors,
	}
	handler := Handler{
		compute: &computerHandler,
		storage: &storageHandler,
		ping:    &pingHandler,
	}
	return &handler
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
