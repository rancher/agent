package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/types"
	"github.com/docker/go-units"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/docker"
	"github.com/rancher/agent/utilities/utils"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/event-subscriber/locks"
	"github.com/rancher/go-rancher/v2"
	"golang.org/x/net/context"
	"gopkg.in/check.v1"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	check.TestingT(t)
}

type ComputeTestSuite struct {
}

var _ = check.Suite(&ComputeTestSuite{})

func (s *ComputeTestSuite) SetUpSuite(c *check.C) {
}

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
	dockerClient := docker.GetClient(constants.DefaultVersion)
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
	dockerClient := docker.GetClient(constants.DefaultVersion)
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
	fields["tmpfs"] = map[string]string{
		"/run": "rw,noexec,nosuid,size=65536k",
	}
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
	dockerClient := docker.GetClient(constants.DefaultVersion)
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
	c.Assert(inspect.HostConfig.Tmpfs, check.DeepEquals, map[string]string{
		"/run": "rw,noexec,nosuid,size=65536k",
	})
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
	dockerClient := docker.GetClient(constants.DefaultVersion)
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

// need docker daemon with higher version than 1.10.3
func (s *ComputeTestSuite) TestNewFieldsExtra(c *check.C) {
	deleteContainer("/c861f990-4472-4fa1-960f-65171b544c28")
	rawEvent := loadEvent("./test_events/instance_activate_basic", c)
	event, _, fields := unmarshalEventAndInstanceFields(rawEvent, c)
	fields["sysctls"] = map[string]string{
		"net.ipv4.ip_forward": "1",
	}
	fields["healthCmd"] = []string{
		"ls",
	}
	fields["healthInterval"] = 5
	fields["healthRetries"] = 3
	fields["healthTimeout"] = 60
	rawEvent = marshalEvent(event, c)
	reply := testEvent(rawEvent, c)
	container, ok := utils.GetFieldsIfExist(reply.Data, "instanceHostMap", "instance", "+data", "dockerContainer")
	if !ok {
		c.Fatal("No id found")
	}
	dockerClient := docker.GetClient(constants.DefaultVersion)
	version, err := dockerClient.ServerVersion(context.Background())
	if err != nil {
		c.Fatal(err)
	}
	if version.Version == "1.12.1" {
		inspect, err := dockerClient.ContainerInspect(context.Background(), container.(types.Container).ID)
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
	}

}

func unmarshalEventAndInstanceFields(rawEvent []byte, c *check.C) (map[string]interface{}, map[string]interface{},
	map[string]interface{}) {
	event := unmarshalEvent(rawEvent, c)
	instance := event["data"].(map[string]interface{})["instanceHostMap"].(map[string]interface{})["instance"].(map[string]interface{})
	fields := instance["data"].(map[string]interface{})["fields"].(map[string]interface{})
	return event, instance, fields
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
	dockerClient := docker.GetClient(constants.DefaultVersion)
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

func (s *ComputeTestSuite) TestInstanceActivateWindowsImage(c *check.C) {
	if runtime.GOOS == "windows" {
		deleteContainer("/c861f990-4472-4fa1-960f-65171b544c26")

		rawEvent := loadEvent("./test_events/instance_activate_windows", c)
		reply := testEvent(rawEvent, c)
		container, ok := utils.GetFieldsIfExist(reply.Data, "instanceHostMap", "instance", "+data", "dockerContainer")
		if !ok {
			c.Fatal("No ID found")
		}
		dockerClient := docker.GetClient(constants.DefaultVersion)
		inspect, err := dockerClient.ContainerInspect(context.Background(), container.(*types.Container).ID)
		if err != nil {
			c.Fatal("Inspect Err")
		}
		c.Check(inspect.Config.Image, check.Equals, "microsoft/iis:latest")
	}
}

func (s *ComputeTestSuite) TestInstanceDeactivateWindowsImage(c *check.C) {
	if runtime.GOOS == "windows" {
		deleteContainer("/c861f990-4472-4fa1-960f-65171b544c26")

		rawEvent := loadEvent("./test_events/instance_activate_windows", c)
		reply := testEvent(rawEvent, c)
		container, ok := utils.GetFieldsIfExist(reply.Data, "instanceHostMap", "instance", "+data", "dockerContainer")
		if !ok {
			c.Fatal("No ID found")
		}
		dockerClient := docker.GetClient(constants.DefaultVersion)
		inspect, err := dockerClient.ContainerInspect(context.Background(), container.(*types.Container).ID)
		if err != nil {
			c.Fatal("Inspect Err")
		}
		c.Check(inspect.Config.Image, check.Equals, "microsoft/iis:latest")

		rawEventDe := loadEvent("./test_events/instance_deactivate_windows", c)
		replyDe := testEvent(rawEventDe, c)
		container, ok = utils.GetFieldsIfExist(replyDe.Data, "instanceHostMap", "instance", "+data", "dockerContainer")
		if !ok {
			c.Fatal("No ID found")
		}
		inspect, err = dockerClient.ContainerInspect(context.Background(), container.(*types.Container).ID)
		if err != nil {
			c.Fatal("Inspect Err")
		}
		c.Check(inspect.State.Status, check.Equals, "exited")
	}
}

func deleteContainer(name string) {
	dockerClient := docker.GetClient(constants.DefaultVersion)
	containerList, _ := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	for _, c := range containerList {
		found := false
		labels := c.Labels
		if labels["io.rancher.container.uuid"] == name[1:] {
			found = true
		}

		for _, cname := range c.Names {
			if name == cname {
				found = true
				break
			}
		}
		if found {
			dockerClient.ContainerKill(context.Background(), c.ID, "KILL")
			for i := 0; i < 10; i++ {
				if inspect, err := dockerClient.ContainerInspect(context.Background(), c.ID); err == nil && inspect.State.Pid == 0 {
					break
				}
				time.Sleep(time.Duration(500) * time.Millisecond)
			}
			dockerClient.ContainerRemove(context.Background(), c.ID, types.ContainerRemoveOptions{})
			RemoveStateFile(c.ID)
		}
	}
}

func RemoveStateFile(id string) {
	if len(id) > 0 {
		contDir := config.ContainerStateDir()
		filePath := path.Join(contDir, id)
		if _, err := os.Stat(filePath); err == nil {
			os.Remove(filePath)
		}
	}
}

func checkStringInArray(array []string, item string) bool {
	for _, str := range array {
		if str == item {
			return true
		}
	}
	return false
}

func loadEvent(eventFile string, c *check.C) []byte {
	file, err := ioutil.ReadFile(eventFile)
	if err != nil {
		c.Fatalf("Error reading event %v", err)
	}
	return file

}

func getInstance(event map[string]interface{}, c *check.C) map[string]interface{} {
	data := event["data"].(map[string]interface{})
	ihm := data["instanceHostMap"].(map[string]interface{})
	instance := ihm["instance"].(map[string]interface{})
	return instance
}

func unmarshalEvent(rawEvent []byte, c *check.C) map[string]interface{} {
	event := map[string]interface{}{}
	err := json.Unmarshal(rawEvent, &event)
	if err != nil {
		c.Fatalf("Error unmarshalling event %v", err)
	}
	return event
}

func marshalEvent(event interface{}, c *check.C) []byte {
	b, err := json.Marshal(event)
	if err != nil {
		c.Fatalf("Error marshalling event %v", err)
	}
	return b
}

func testEvent(rawEvent []byte, c *check.C) *client.Publish {
	apiClient, mockPublish := newTestClient()
	workers := make(chan *Worker, 1)
	worker := &Worker{}
	worker.DoWork(rawEvent, GetHandlers(), apiClient, workers)
	return mockPublish.publishedResponse
}

func newTestClient() (*client.RancherClient, *mockPublishOperations) {
	mock := &mockPublishOperations{}
	return &client.RancherClient{
		Publish: mock,
	}, mock
}

/*
type PublishOperations interface {
	List(opts *ListOpts) (*PublishCollection, error)
	Create(opts *Publish) (*Publish, error)
	Update(existing *Publish, updates interface{}) (*Publish, error)
	ById(id string) (*Publish, error)
	Delete(container *Publish) error
}
*/
type mockPublishOperations struct {
	publishedResponse *client.Publish
}

func (m *mockPublishOperations) Create(publish *client.Publish) (*client.Publish, error) {
	m.publishedResponse = publish
	return publish, nil
}

func (m *mockPublishOperations) List(publish *client.ListOpts) (*client.PublishCollection, error) {
	return nil, fmt.Errorf("Mock not implemented.")
}

func (m *mockPublishOperations) Update(existing *client.Publish, updates interface{}) (*client.Publish, error) {
	return nil, fmt.Errorf("Mock not implemented.")
}

func (m *mockPublishOperations) ById(id string) (*client.Publish, error) { // golint_ignore
	return nil, fmt.Errorf("Mock not implemented.")
}

func (m *mockPublishOperations) Delete(existing *client.Publish) error {
	return fmt.Errorf("Mock not implemented.")
}

type Worker struct {
}

func (w *Worker) DoWork(rawEvent []byte, eventHandlers map[string]revents.EventHandler, apiClient *client.RancherClient,
	workers chan *Worker) {
	defer func() { workers <- w }()

	event := &revents.Event{}
	err := json.Unmarshal(rawEvent, &event)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"err": err,
		}).Error("Error unmarshalling event")
		return
	}

	if event.Name != "ping" {
		logrus.WithFields(logrus.Fields{
			"event": string(rawEvent[:]),
		}).Debug("Processing event.")
	}

	unlocker := locks.Lock(event.ResourceID)
	if unlocker == nil {
		logrus.WithFields(logrus.Fields{
			"resourceId": event.ResourceID,
		}).Debug("Resource locked. Dropping event")
		return
	}
	defer unlocker.Unlock()

	if fn, ok := eventHandlers[event.Name]; ok {
		err = fn(event, apiClient)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"eventName":  event.Name,
				"eventId":    event.ID,
				"resourceId": event.ResourceID,
				"err":        err,
			}).Error("Error processing event")

			reply := &client.Publish{
				Name:                 event.ReplyTo,
				PreviousIds:          []string{event.ID},
				Transitioning:        "error",
				TransitioningMessage: err.Error(),
			}
			_, err := apiClient.Publish.Create(reply)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"err": err,
				}).Error("Error sending error-reply")
			}
		}
	} else {
		logrus.WithFields(logrus.Fields{
			"eventName": event.Name,
		}).Warn("No event handler registered for event")
	}
}
