//+build !windows

package handlers

import (
	"bufio"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-units"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/docker"
	"github.com/rancher/agent/utilities/utils"
	"golang.org/x/net/context"
	"gopkg.in/check.v1"
	"os"
	"strings"
	"time"
)

type mt map[string]interface{}

func (s *ComputeTestSuite) TestMillCpuReservation(c *check.C) {
	deleteContainer("/c861f990-4472-4fa1-960f-65171b544c28")

	rawEvent := loadEvent("./test_events/instance_activate_basic", c)
	event, instance, fields := unmarshalEventAndInstanceFields(rawEvent, c)

	instance["milliCpuReservation"] = 200
	fields["cpuShares"] = 100
	rawEvent = marshalEvent(event, c)
	reply := testEvent(rawEvent, c)

	container, ok := utils.GetFieldsIfExist(reply.Data, "instanceHostMap", "instance", "+data", "dockerContainer")
	if !ok {
		c.Fatal("No id found")
	}
	dockerClient := docker.GetClient(docker.DefaultVersion)
	inspect, err := dockerClient.ContainerInspect(context.Background(), container.(types.Container).ID)
	if err != nil {
		c.Fatal("Inspect Err")
	}

	// Value should be 20% of 1024, rounded down
	c.Assert(inspect.HostConfig.CPUShares, check.Equals, int64(204))
}

func (s *ComputeTestSuite) TestMemoryReservation(c *check.C) {
	deleteContainer("/c861f990-4472-4fa1-960f-65171b544c28")

	rawEvent := loadEvent("./test_events/instance_activate_basic", c)
	event, instance, _ := unmarshalEventAndInstanceFields(rawEvent, c)

	instance["memoryReservation"] = 4194304 // 4MB, the minimum
	rawEvent = marshalEvent(event, c)
	reply := testEvent(rawEvent, c)

	container, ok := utils.GetFieldsIfExist(reply.Data, "instanceHostMap", "instance", "+data", "dockerContainer")
	if !ok {
		c.Fatal("No id found")
	}
	dockerClient := docker.GetClient(docker.DefaultVersion)
	inspect, err := dockerClient.ContainerInspect(context.Background(), container.(types.Container).ID)
	if err != nil {
		c.Fatal("Inspect Err")
	}

	c.Assert(inspect.HostConfig.MemoryReservation, check.Equals, int64(4194304))
}

func (s *ComputeTestSuite) TestNewFields(c *check.C) {
	deleteContainer("/c861f990-4472-4fa1-960f-65171b544c28")
	rawEvent := loadEvent("./test_events/instance_activate_basic", c)
	event, _, fields := unmarshalEventAndInstanceFields(rawEvent, c)
	fields["privileged"] = true
	fields["blkioWeight"] = 100
	fields["cpuPeriod"] = 100000
	fields["cpuQuota"] = 50000
	fields["cpuSetMems"] = "0"
	fields["kernelMemory"] = 10000000
	fields["memory"] = 10000000
	fields["memorySwappiness"] = 50
	fields["oomKillDisable"] = true
	fields["oomScoreAdj"] = 500
	fields["shmSize"] = 67108864
	fields["groupAdd"] = []string{"root"}
	fields["uts"] = "host"
	fields["ipcMode"] = "host"
	fields["stopSignal"] = "SIGTERM"
	fields["ulimits"] = []map[string]interface{}{
		{
			"name": "cpu",
			"hard": 100000,
			"soft": 100000,
		},
	}

	rawEvent = marshalEvent(event, c)
	reply := testEvent(rawEvent, c)
	container, ok := utils.GetFieldsIfExist(reply.Data, "instanceHostMap", "instance", "+data", "dockerContainer")
	if !ok {
		c.Fatal("No id found")
	}
	dockerClient := docker.GetClient(docker.DefaultVersion)
	inspect, err := dockerClient.ContainerInspect(context.Background(), container.(types.Container).ID)
	if err != nil {
		c.Fatal("Inspect Err")
	}

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

func (s *ComputeTestSuite) TestDNSFields(c *check.C) {
	// this test aims to verify that if the dnsSearch is set to rancher.internal, we should add dnssearch from host to
	// containers
	deleteContainer("/c861f990-4472-4fa1-960f-65171b544c28")
	rawEvent := loadEvent("./test_events/instance_activate_basic", c)
	event, _, fields := unmarshalEventAndInstanceFields(rawEvent, c)
	fields["dnsSearch"] = []string{"rancher.internal"}
	rawEvent = marshalEvent(event, c)
	reply := testEvent(rawEvent, c)
	container, ok := utils.GetFieldsIfExist(reply.Data, "instanceHostMap", "instance", "+data", "dockerContainer")
	if !ok {
		c.Fatal("No id found")
	}
	dockerClient := docker.GetClient(docker.DefaultVersion)
	inspect, err := dockerClient.ContainerInspect(context.Background(), container.(types.Container).ID)
	if err != nil {
		c.Fatal("Inspect Err")
	}
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
func (s *ComputeTestSuite) TestNewFieldsExtra(c *check.C) {
	dockerClient := docker.GetClient(docker.DefaultVersion)
	version, err := dockerClient.ServerVersion(context.Background())
	if err != nil {
		c.Fatal(err)
	}
	if version.Version == "1.12.1" {
		deleteContainer("/c861f990-4472-4fa1-960f-65171b544c28")
		rawEvent := loadEvent("./test_events/instance_activate_basic", c)
		event, _, fields := unmarshalEventAndInstanceFields(rawEvent, c)
		fields["sysctls"] = map[string]string{
			"net.ipv4.ip_forward": "1",
		}
		fields["tmpfs"] = map[string]string{
			"/run": "rw,noexec,nosuid,size=65536k",
		}
		fields["healthCmd"] = []string{
			"ls",
		}
		fields["usernsMode"] = "host"
		fields["healthInterval"] = 5
		fields["healthRetries"] = 3
		fields["healthTimeout"] = 60
		rawEvent = marshalEvent(event, c)
		reply := testEvent(rawEvent, c)
		cont, ok := utils.GetFieldsIfExist(reply.Data, "instanceHostMap", "instance", "+data", "dockerContainer")
		if !ok {
			c.Fatal("No id found")
		}
		inspect, err := dockerClient.ContainerInspect(context.Background(), cont.(types.Container).ID)
		if err != nil {
			c.Fatal("Inspect Err")
		}
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

func (s *ComputeTestSuite) TestInstanceActivateAgent(c *check.C) {
	constants.ConfigOverride["CONFIG_URL"] = "https://localhost:1234/a/path"
	deleteContainer("/c861f990-4472-4fa1-960f-65171b544c28")

	rawEvent := loadEvent("./test_events/instance_activate_agent_instance", c)
	reply := testEvent(rawEvent, c)
	container, ok := utils.GetFieldsIfExist(reply.Data, "instanceHostMap", "instance", "+data", "dockerContainer")
	if !ok {
		c.Fatal("No id found")
	}
	dockerClient := docker.GetClient(docker.DefaultVersion)
	inspect, err := dockerClient.ContainerInspect(context.Background(), container.(types.Container).ID)
	if err != nil {
		c.Fatal("Inspect Err")
	}
	port := config.APIProxyListenPort()
	ok1 := checkStringInArray(inspect.Config.Env, "CATTLE_CONFIG_URL_SCHEME=https")
	ok2 := checkStringInArray(inspect.Config.Env, "CATTLE_CONFIG_URL_PATH=/a/path")
	ok3 := checkStringInArray(inspect.Config.Env, fmt.Sprintf("CATTLE_CONFIG_URL_PORT=%v", port))
	c.Assert(ok1, check.Equals, true)
	c.Assert(ok2, check.Equals, true)
	c.Assert(ok3, check.Equals, true)
}
