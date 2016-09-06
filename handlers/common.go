package handlers

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/types"
	goUUID "github.com/nu7hatch/gouuid"
	"github.com/rancher/agent/core/hostInfo"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/docker"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/client"
	"golang.org/x/net/context"
	"runtime"
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
		"compute.instance.activate":   handler.compute.InstanceActivate,
		"compute.instance.deactivate": handler.compute.InstanceDeactivate,
		"compute.instance.force.stop": handler.compute.InstanceForceStop,
		"compute.instance.inspect":    handler.compute.InstanceInspect,
		"compute.instance.pull":       handler.compute.InstancePull,
		"compute.instance.remove":     handler.compute.InstanceRemove,
		"storage.image.activate":      handler.storage.ImageActivate,
		"storage.volume.activate":     handler.storage.VolumeActivate,
		"storage.volume.deactivate":   handler.storage.VolumeDeactivate,
		"storage.volume.remove":       handler.storage.VolumeRemove,
		"delegate.request":            handler.delegate.DelegateRequest,
		"ping":                        handler.ping.Ping,
		"config.update":               handler.configUpdate.ConfigUpdate,
	}
}

func reply(replyData map[string]interface{}, event *revents.Event, cli *client.RancherClient) error {
	if replyData == nil {
		replyData = make(map[string]interface{})
	}

	reply := &client.Publish{
		ResourceId:    event.ResourceID,
		PreviousIds:   []string{event.ID},
		ResourceType:  event.ResourceType,
		Name:          event.ReplyTo,
		Data:          replyData,
		Time:          time.Now().UnixNano() / int64(time.Millisecond),
		Resource:      client.Resource{Id: getUUID()},
		PreviousNames: []string{event.Name},
	}

	logrus.Infof("Reply: %+v", reply)
	err := publishReply(reply, cli)
	if err != nil {
		return fmt.Errorf("Error sending reply %v: %v", event.ID, err)
	}
	return nil
}

func initializeHandlers() *Handler {
	client := docker.GetClient(constants.DefaultVersion)
	info := types.Info{}
	version := types.Version{}
	// initialize the info and version so we don't have to call docker API every time a ping request comes
	for i := 0; i < 10; i++ {
		in, err := client.Info(context.Background())
		if err == nil {
			info = in
			break
		}
		time.Sleep(time.Duration(1) * time.Second)
	}
	for i := 0; i < 10; i++ {
		v, err := client.ServerVersion(context.Background())
		if err == nil {
			version = v
			break
		}
		time.Sleep(time.Duration(1) * time.Second)
	}
	Cadvisor := hostInfo.CadvisorAPIClient{
		DataGetter: hostInfo.CadvisorDataGetter{
			URL: fmt.Sprintf("%v%v:%v/api/%v", "http://", config.CadvisorIP(), config.CadvisorPort(), "v1.2"),
		},
	}
	Collectors := []hostInfo.Collector{
		hostInfo.CPUCollector{
			Cadvisor:   Cadvisor,
			DataGetter: hostInfo.CPUDataGetter{},
			GOOS:       runtime.GOOS,
		},
		hostInfo.DiskCollector{
			Cadvisor:   Cadvisor,
			Unit:       1048576,
			DataGetter: hostInfo.DiskDataGetter{},
			InfoData:   model.InfoData{
				Info: info,
				Version: version,
			},
		},
		hostInfo.IopsCollector{
			GOOS: runtime.GOOS,
		},
		hostInfo.MemoryCollector{
			Unit:       1024.00,
			DataGetter: hostInfo.MemoryDataGetter{},
			GOOS:       runtime.GOOS,
		},
		hostInfo.OSCollector{
			DataGetter: hostInfo.OSDataGetter{},
			GOOS:       runtime.GOOS,
			InfoData:   model.InfoData{
				Info: info,
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
	child := map[string]interface{}{
		"resourceId":    event.ResourceID,
		"previousIds":   []string{event.ID},
		"resourceType":  event.ResourceType,
		"name":          event.ReplyTo,
		"data":          replyData,
		"id":            getUUID(),
		"time":          time.Now().UnixNano() / int64(time.Millisecond),
		"previousNames": []string{event.Name},
	}
	reply := &client.Publish{
		ResourceId:    parent.ResourceID,
		PreviousIds:   []string{parent.ID},
		ResourceType:  parent.ResourceType,
		Name:          parent.ReplyTo,
		Data:          child,
		Time:          time.Now().UnixNano() / int64(time.Millisecond),
		Resource:      client.Resource{Id: getUUID()},
		PreviousNames: []string{parent.Name},
	}
	if parent.ReplyTo == "" {
		return nil
	}
	logrus.Infof("Reply: %+v", reply)
	err := publishReply(reply, cli)
	if err != nil {
		return fmt.Errorf("Error sending reply %v: %v", event.ID, err)
	}
	return nil
}

func getUUID() string {
	uuid := ""
	newUUID, err1 := goUUID.NewV4()
	if err1 != nil {
		logrus.Error(err1)
	} else {
		uuid = newUUID.String()
	}
	return uuid
}

func publishReply(reply *client.Publish, apiClient *client.RancherClient) error {
	_, err := apiClient.Publish.Create(reply)
	if err != nil {
		logrus.Error(err)
	}
	return err
}
