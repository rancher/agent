package handlers

import (
	"github.com/docker/docker/client"
	"github.com/rancher/agent/host_info"
)

type Handler struct {
	compute *ComputeHandler
	storage *StorageHandler
	ping    *PingHandler
}

type ComputeHandler struct {
	dockerClient *client.Client
}

type PingHandler struct {
	dockerClient *client.Client
	collectors   []hostInfo.Collector
}

type StorageHandler struct {
	dockerClient *client.Client
}
