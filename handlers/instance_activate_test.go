//+build !windows

package handlers

import (
	"github.com/docker/go-units"
	v2 "github.com/rancher/go-rancher/v2"
	"gopkg.in/check.v1"
	"github.com/rancher/agent/utils"
	"os"
	"bufio"
	"strings"
	"github.com/docker/docker/api/types/container"
	"golang.org/x/net/context"
	"time"
	"fmt"
)

type mt map[string]interface{}

func (s *EventTestSuite) TestMillCpuReservation(c *check.C) {
	deleteContainer("85db87bf-cb14-4643-9e7d-a13e3e77a991")

	var request v2.DeploymentSyncRequest
	event := getDeploymentSyncRequest("./test_events/deployment_sync_request", &request, c)
	c.Assert(request.Containers, check.HasLen, 1)

	request.Containers[0].MilliCpuReservation = 200
	request.Containers[0].CpuShares = 100
	event.Data["deploymentSyncRequest"] = request
	rawEvent := marshalEvent(event, c)
	reply := testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)

	inspect := getDockerInspect(reply, c)

	// Value should be 20% of 1024, rounded down
	c.Assert(inspect.HostConfig.CPUShares, check.Equals, int64(204))
}

func (s *EventTestSuite) TestMemoryReservation(c *check.C) {
	deleteContainer("85db87bf-cb14-4643-9e7d-a13e3e77a991")

	var request v2.DeploymentSyncRequest
	event := getDeploymentSyncRequest("./test_events/deployment_sync_request", &request, c)
	c.Assert(request.Containers, check.HasLen, 1)

	request.Containers[0].MemoryReservation = 4194304 // 4MB, the minimum

	event.Data["deploymentSyncRequest"] = request
	rawEvent := marshalEvent(event, c)
	reply := testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)

	inspect := getDockerInspect(reply, c)

	c.Assert(inspect.HostConfig.MemoryReservation, check.Equals, int64(4194304))
}

func (s *EventTestSuite) TestLabelOverride(c *check.C) {
	deleteContainer("85db87bf-cb14-4643-9e7d-a13e3e77a991")

	var request v2.DeploymentSyncRequest
	event := getDeploymentSyncRequest("./test_events/deployment_sync_request", &request, c)
	c.Assert(request.Containers, check.HasLen, 1)

	request.Containers[0].Labels = map[string]interface{}{
		"io.rancher.container.uuid": "111",
		"foo": "bar",
	}

	event.Data["deploymentSyncRequest"] = request
	rawEvent := marshalEvent(event, c)
	reply := testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)

	inspect := getDockerInspect(reply, c)

	expectedLabels := map[string]string{
		"foo": "bar",
		"io.rancher.container.uuid":        request.Containers[0].Uuid,
		"io.rancher.container.name":        request.Containers[0].Name,
		"io.rancher.container.pull_image":  "always",
		"io.rancher.container.mac_address": "02:0c:9e:04:d1:39",
	}
	c.Assert(inspect.Config.Labels, check.DeepEquals, expectedLabels)
}

func (s *EventTestSuite) TestDockerFields(c *check.C) {
	deleteContainer("85db87bf-cb14-4643-9e7d-a13e3e77a991")

	var request v2.DeploymentSyncRequest
	event := getDeploymentSyncRequest("./test_events/deployment_sync_request", &request, c)
	c.Assert(request.Containers, check.HasLen, 1)

	request.Containers[0].Privileged = true
	request.Containers[0].BlkioWeight = 100
	request.Containers[0].CpuPeriod = 100000
	request.Containers[0].CpuQuota = 50000
	request.Containers[0].CpuSetMems = "0"
	request.Containers[0].KernelMemory = 10000000
	request.Containers[0].Memory = 10000000
	request.Containers[0].MemorySwappiness = 50
	request.Containers[0].OomKillDisable = true
	request.Containers[0].OomScoreAdj = 500
	request.Containers[0].ShmSize = 67108864
	request.Containers[0].GroupAdd = []string{"root"}
	request.Containers[0].Uts = "host"
	request.Containers[0].IpcMode = "host"
	request.Containers[0].StopSignal = "SIGTERM"
	request.Containers[0].Ulimits = []v2.Ulimit{
		{
			Name: "cpu",
			Hard: 100000,
			Soft: 100000,
		},
	}

	event.Data["deploymentSyncRequest"] = request
	rawEvent := marshalEvent(event, c)
	reply := testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)

	inspect := getDockerInspect(reply, c)

	c.Assert(inspect.HostConfig.BlkioWeight, check.Equals, uint16(100))
	c.Assert(inspect.HostConfig.CPUPeriod, check.Equals, int64(100000))
	c.Assert(inspect.HostConfig.CPUQuota, check.Equals, int64(50000))
	c.Assert(inspect.HostConfig.CpusetMems, check.Equals, "0")
	c.Assert(inspect.HostConfig.KernelMemory, check.Equals, int64(10000000))
	c.Assert(inspect.HostConfig.Memory, check.Equals, int64(10000000))
	c.Assert(*(inspect.HostConfig.MemorySwappiness), check.Equals, int64(50))
	c.Assert(*(inspect.HostConfig.OomKillDisable), check.Equals, true)
	c.Assert(inspect.HostConfig.OomScoreAdj, check.Equals, 500)
	c.Assert(inspect.HostConfig.ShmSize, check.Equals, int64(67108864))
	c.Assert(inspect.HostConfig.GroupAdd, check.DeepEquals, []string{"root"})
	c.Assert(string(inspect.HostConfig.UTSMode), check.Equals, "host")
	c.Assert(string(inspect.HostConfig.IpcMode), check.Equals, "host")
	c.Assert(inspect.Config.StopSignal, check.Equals, "SIGTERM")
	ulimits := []units.Ulimit{
		{
			Name: "cpu",
			Hard: int64(100000),
			Soft: int64(100000),
		},
	}
	c.Assert(*(inspect.HostConfig.Ulimits[0]), check.DeepEquals, ulimits[0])
}

func (s *EventTestSuite) TestDNSFields(c *check.C) {
	// this test aims to verify that if the dnsSearch is set to rancher.internal, we should add dnssearch from host to
	// containers
	deleteContainer("85db87bf-cb14-4643-9e7d-a13e3e77a991")

	var request v2.DeploymentSyncRequest
	event := getDeploymentSyncRequest("./test_events/deployment_sync_request", &request, c)
	c.Assert(request.Containers, check.HasLen, 1)

	request.Containers[0].DnsSearch = []string{"rancher.internal"}

	event.Data["deploymentSyncRequest"] = request
	rawEvent := marshalEvent(event, c)
	reply := testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)

	inspect := getDockerInspect(reply, c)

	file, err := os.Open("/etc/resolv.conf")
	if err != nil {
		c.Fatal(err)
	}
	defer file.Close()
	dnsSearch := []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		s := []string{}
		if strings.HasPrefix(line, "search") {
			// in case multiple search lines
			// respect the last one
			s = strings.Split(line, " ")[1:]
			for i := range s {
				search := s[len(s)-i-1]
				if !utils.SearchInList(dnsSearch, search) {
					dnsSearch = append([]string{search}, dnsSearch...)
				}
			}
		}
	}
	c.Assert(inspect.HostConfig.DNSSearch, check.DeepEquals, append(dnsSearch, "rancher.internal"))
}

// need docker daemon with version 1.12.1
func (s *EventTestSuite) TestDockerFieldsExtra(c *check.C) {
	dockerClient := utils.GetRuntimeClient("docker", utils.DefaultVersion)
	version, err := dockerClient.ServerVersion(context.Background())
	if err != nil {
		c.Fatal(err)
	}
	if version.Version == "1.12.1" {
		deleteContainer("85db87bf-cb14-4643-9e7d-a13e3e77a991")

		var request v2.DeploymentSyncRequest
		event := getDeploymentSyncRequest("./test_events/deployment_sync_request", &request, c)
		c.Assert(request.Containers, check.HasLen, 1)

		request.Containers[0].Sysctls = map[string]interface{}{
			"net.ipv4.ip_forward": "1",
		}
		request.Containers[0].Tmpfs = map[string]interface{}{
			"net.ipv4.ip_forward": "1",
		}
		request.Containers[0].HealthCmd = []string{"ls"}
		request.Containers[0].UsernsMode = "host"
		request.Containers[0].HealthInterval = 5
		request.Containers[0].HealthTimeout = 60
		request.Containers[0].HealthRetries = 3


		event.Data["deploymentSyncRequest"] = request
		rawEvent := marshalEvent(event, c)
		reply := testEvent(rawEvent, c)

		c.Assert(reply.Transitioning != "error", check.Equals, true)

		inspect := getDockerInspect(reply, c)

		c.Assert(inspect.HostConfig.Sysctls, check.DeepEquals, map[string]string{
			"net.ipv4.ip_forward": "1",
		})
		c.Assert(inspect.Config.Healthcheck.Test, check.DeepEquals, []string{"ls"})
		c.Assert(inspect.Config.Healthcheck.Retries, check.Equals, 3)
		c.Assert(inspect.Config.Healthcheck.Timeout, check.Equals, time.Duration(60)*time.Second)
		c.Assert(inspect.Config.Healthcheck.Interval, check.Equals, time.Duration(5)*time.Second)
		c.Assert(inspect.HostConfig.Tmpfs, check.DeepEquals, map[string]string{
			"/run": "rw,noexec,nosuid,size=65536k",
		})
		c.Assert(inspect.HostConfig.UsernsMode, check.Equals, container.UsernsMode("host"))
	}
}

// need docker daemon with version 1.13.1
func (s *EventTestSuite) TestNewFieldsExtra_1_13(c *check.C) {
	dockerClient := utils.GetRuntimeClient("docker", utils.DefaultVersion)
	version, err := dockerClient.ServerVersion(context.Background())
	if err != nil {
		c.Fatal(err)
	}
	if version.Version == "1.13.1" {
		deleteContainer("85db87bf-cb14-4643-9e7d-a13e3e77a991")

		var request v2.DeploymentSyncRequest
		getDeploymentSyncRequest("./test_events/deployment_sync_request", &request, c)
		c.Assert(request.Containers, check.HasLen, 1)

		// TODO: add init tests
	}
}

func (s *EventTestSuite) TestInstanceActivateAgent(c *check.C) {
	utils.ConfigOverride["CONFIG_URL"] = "https://localhost:1234/a/path"

	deleteContainer("85db87bf-cb14-4643-9e7d-a13e3e77a991")

	var request v2.DeploymentSyncRequest
	event := getDeploymentSyncRequest("./test_events/deployment_sync_request", &request, c)
	c.Assert(request.Containers, check.HasLen, 1)

	request.Containers[0].AgentId = "1a1"

	event.Data["deploymentSyncRequest"] = request
	rawEvent := marshalEvent(event, c)
	reply := testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)

	inspect := getDockerInspect(reply, c)

	port := utils.APIProxyListenPort()
	ok1 := checkStringInArray(inspect.Config.Env, "CATTLE_CONFIG_URL_SCHEME=https")
	ok2 := checkStringInArray(inspect.Config.Env, "CATTLE_CONFIG_URL_PATH=/a/path")
	ok3 := checkStringInArray(inspect.Config.Env, fmt.Sprintf("CATTLE_CONFIG_URL_PORT=%v", port))
	c.Assert(ok1, check.Equals, true)
	c.Assert(ok2, check.Equals, true)
	c.Assert(ok3, check.Equals, true)
}

func (s *EventTestSuite) TestInstanceActivateNoName(c *check.C) {
	deleteContainer("85db87bf-cb14-4643-9e7d-a13e3e77a991")

	var request v2.DeploymentSyncRequest
	event := getDeploymentSyncRequest("./test_events/deployment_sync_request", &request, c)
	c.Assert(request.Containers, check.HasLen, 1)

	request.Containers[0].Name = ""

	event.Data["deploymentSyncRequest"] = request
	rawEvent := marshalEvent(event, c)
	reply := testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)

	inspect := getDockerInspect(reply, c)
	c.Assert(inspect.ContainerJSONBase.Name, check.Equals, "/r-" + request.Containers[0].Uuid)
}

func (s *EventTestSuite) TestInstanceActivateBasic(c *check.C) {
	deleteContainer("85db87bf-cb14-4643-9e7d-a13e3e77a991")

	var request v2.DeploymentSyncRequest
	event := getDeploymentSyncRequest("./test_events/deployment_sync_request", &request, c)
	c.Assert(request.Containers, check.HasLen, 1)

	request.Containers[0].PublicEndpoints = []v2.PublicEndpoint{
		{
			PublicPort: 10000,
			PrivatePort: 10000,
			Protocol: "tcp",
		},
		{
			PublicPort: 10001,
			PrivatePort: 10000,
			Protocol: "udp",
		},
		{
			PublicPort: 10002,
			PrivatePort: 10000,
			Protocol: "udp",
			BindIpAddress: "127.0.0.1",
		},
	}
	request.Containers[0].CpuSet = "0,1"
	request.Containers[0].ReadOnly = true
	request.Containers[0].Memory = 12000000
	request.Containers[0].MemorySwap = 16000000
	request.Containers[0].ExtraHosts = []string{"host:1.1.1.1", "b:2.2.2.2"}
	request.Containers[0].PidMode = "host"
	request.Containers[0].LogConfig = &v2.LogConfig{
		Driver: "json-file",
		Config: map[string]interface{}{
			"max-size": "10",
		},
	}
	request.Containers[0].SecurityOpt = []string{"label:foo", "label:bar"}
	request.Containers[0].WorkingDir = "/home"
	request.Containers[0].EntryPoint = []string{"./sleep.sh"}
	request.Containers[0].Tty = true
	request.Containers[0].StdinOpen = true
	request.Containers[0].DomainName = "rancher.io"
	request.Containers[0].Devices = []string{"/dev/null:/dev/xnull", "/dev/random:/dev/xrandom:rw"}
	request.Containers[0].Dns = []string{"1.2.3.4", "8.8.8.8"}
	request.Containers[0].DnsSearch = []string{"5.6.7.8", "7.7.7.7"}
	request.Containers[0].CapAdd = []string{"MKNOD", "SYS_ADMIN"}
	request.Containers[0].CapDrop = []string{"MKNOD", "SYS_ADMIN"}
	request.Containers[0].Privileged = true
	request.Containers[0].RestartPolicy = &v2.RestartPolicy{
		Name: "always",
		MaximumRetryCount: 2,
	}

	event.Data["deploymentSyncRequest"] = request
	rawEvent := marshalEvent(event, c)
	reply := testEvent(rawEvent, c)

	c.Assert(reply.Transitioning != "error", check.Equals, true)

	c.Assert(inspect.ContainerJSONBase.Name, check.Equals, "/r-" + request.Containers[0].Uuid)
}