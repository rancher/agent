package handlers

import (
	"fmt"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	goUUID "github.com/nu7hatch/gouuid"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/rancher/agent/core/hostInfo"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utils/docker"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v2"
	"golang.org/x/net/context"
)

type Handler struct {
	compute      *ComputeHandler
	storage      *StorageHandler
	configUpdate *ConfigUpdateHandler
	ping         *PingHandler
}

func GetHandlers() (map[string]revents.EventHandler, error) {
	handler := initializeHandlers()
	info, err := handler.compute.dockerClient.Info(context.Background())
	if err != nil {
		return nil, err
	}

	docker.SerializeCompute = (info.Driver == "devicemapper")
	logrus.Infof("Serialize compute requests: %v, driver: %s", docker.SerializeCompute, info.Driver)

	return map[string]revents.EventHandler{
		"compute.instance.activate":   cleanLog(logRequest(handler.compute.InstanceActivate)),
		"compute.instance.deactivate": docker.SerializeHandler(cleanLog(logRequest(handler.compute.InstanceDeactivate))),
		"compute.instance.force.stop": docker.SerializeHandler(cleanLog(logRequest(handler.compute.InstanceForceStop))),
		"compute.instance.inspect":    cleanLog(logRequest(handler.compute.InstanceInspect)),
		"compute.instance.pull":       cleanLog(logRequest(handler.compute.InstancePull)),
		"compute.instance.remove":     docker.SerializeHandler(cleanLog(logRequest(handler.compute.InstanceRemove))),
		"storage.volume.remove":       cleanLog(logRequest(handler.storage.VolumeRemove)),
		"ping":                        cleanLog(handler.ping.Ping),
		"config.update":               cleanLog(logRequest(handler.configUpdate.ConfigUpdate)),
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
			logrus.WithFields(logrus.Fields{"err": err}).Debug("Verbose error message")
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
	client := docker.GetClient(docker.DefaultVersion)
	clientWithTimeout, err := docker.NewEnvClientWithTimeout(time.Duration(2) * time.Second)
	if err != nil {
		logrus.Errorf("Err: %v. Can not initialize docker client. Exiting go-agent", err)
	}
	clientWithTimeout.UpdateClientVersion(docker.DefaultVersion)
	info := types.Info{}
	version := types.Version{}
	flags := [2]bool{}
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
	// if we can't get the initialization data the program should exit
	if !flags[0] || !flags[1] {
		logrus.Fatalf("Failed to initialize handlers. Exiting go-agent")
		os.Exit(1)
	}
	storageCache := cache.New(5*time.Minute, 30*time.Second)
	cache := cache.New(5*time.Minute, 30*time.Second)
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
		hostInfo.KeyCollector{},
		hostInfo.CloudProviderCollector{},
	}
	computerHandler := ComputeHandler{
		dockerClientWithTimeout: clientWithTimeout,
		dockerClient:            client,
		infoData: model.InfoData{
			Info:    info,
			Version: version,
		},
		memCache: cache,
	}
	storageHandler := StorageHandler{
		dockerClient: client,
		cache:        storageCache,
	}
	pingHandler := PingHandler{
		dockerClient: clientWithTimeout,
		collectors:   Collectors,
	}
	configHandler := ConfigUpdateHandler{}
	handler := Handler{
		compute:      &computerHandler,
		storage:      &storageHandler,
		ping:         &pingHandler,
		configUpdate: &configHandler,
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
