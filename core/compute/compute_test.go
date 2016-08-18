package compute

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/blkiodev"
	"github.com/docker/engine-api/types/container"
	"github.com/nu7hatch/gouuid"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/docker"
	"github.com/rancher/agent/utilities/utils"
	"golang.org/x/net/context"
	"gopkg.in/check.v1"
	"os"
	"path"
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

/*
// Load the event to a byte array from the specified file
rawEvent := loadEvent("../test_events/instance_activate", c)

// Optional: you can unmarshal, modify, and marshal the event data if you need to. This is equivalent to the "pre"
// functions in python-agent
event := unmarshalEvent(rawEvent, c)
instance := getInstance(event, c)
event["replyTo"] = "new-reply-to"
instance["name"] = "new-name"
rawEvent = marshalEvent(event, c)

// Run the event through the framework
reply := testEvent(rawEvent, c)

// Assert whatever you need to on the reply event. This is equivalent to the "post" functions in python-agent
c.Assert(reply.Name, check.Equals, "new-reply-to")
// As an example, once you implement some more logic, you could verify that the reply has the instance name as "new-name"
*/
func (s *ComputeTestSuite) TestMultiNicsPickMac(c *check.C) {
	instance := model.Instance{}
	instance.Nics = []model.Nic{
		model.Nic{
			MacAddress:   "02:03:04:05:06:07",
			DeviceNumber: 0,
		},
		model.Nic{
			MacAddress:   "02:03:04:05:06:09",
			DeviceNumber: 1,
		},
	}
	config := container.Config{}
	setupMacAndIP(&instance, &config, true, true)
	c.Assert(config.MacAddress, check.Equals, "02:03:04:05:06:07")
}

func (s *ComputeTestSuite) TestDefaultDisk(c *check.C) {
	device := "/dev/mock"
	instance := model.Instance{}
	instance.Data = make(map[string]interface{})
	instance.Data["fields"] = map[string]interface{}{
		"blkioDeviceOptions": map[string]interface{}{
			"DEFAULT_DISK": map[string]interface{}{
				"readIops": 10.0,
			},
		},
	}
	hostConfig := container.HostConfig{}
	setupDeviceOptionsTest(&hostConfig, &instance, device)
	c.Assert(*hostConfig.BlkioDeviceReadIOps[0], check.DeepEquals, blkiodev.ThrottleDevice{Path: device, Rate: 10})
	hostConfig = container.HostConfig{}
	device = ""
	setupDeviceOptionsTest(&hostConfig, &instance, "")
	c.Assert(hostConfig.BlkioDeviceReadIOps, check.HasLen, 0)
}

func (s *ComputeTestSuite) TestNoLabelField(c *check.C) {
	deleteContainer("/no-label-test")
	client := docker.DefaultClient
	_, err := client.ImagePull(context.Background(), "ibuildthecloud/helloworld:latest", types.ImagePullOptions{})
	if err != nil {
		c.Fatal(err)
	}
	time.Sleep(time.Duration(5) * time.Second)
	config := container.Config{Image: "ibuildthecloud/helloworld:latest"}
	resp, err := client.ContainerCreate(context.Background(), &config, nil, nil, "no-label-test")
	if err != nil {
		c.Fatal(err)
	}
	err = client.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{})
	if err != nil {
		c.Fatal(err)
	}
	instance := model.Instance{
		UUID:       "irrelevant",
		ExternalID: resp.ID,
	}
	container := utils.GetContainer(client, &instance, false)
	containers := []map[string]interface{}{}

	container.Labels = map[string]string{}
	containers = utils.AddContainer("running", container, containers)
	c.Assert(containers[0]["uuid"], check.Equals, "no-label-test")
	c.Assert(containers[0]["systemContainer"], check.Equals, "")

	containers = []map[string]interface{}{}
	container.Labels = map[string]string{}
	containers = utils.AddContainer("running", container, containers)
	c.Assert(containers[0]["uuid"], check.Equals, "no-label-test")
	c.Assert(containers[0]["systemContainer"], check.Equals, "")
}

func (s *ComputeTestSuite) TestDefaultValue(c *check.C) {
	varName, _ := uuid.NewV4()
	cattleVarName := fmt.Sprintf("CATTLE_%v", varName)
	def := "defaulted"
	actual := config.DefaultValue(varName.String(), def)
	c.Assert(def, check.Equals, actual)

	actual = config.DefaultValue(varName.String(), "")
	c.Assert(actual, check.Equals, "")

	os.Setenv(cattleVarName, "")
	actual = config.DefaultValue(varName.String(), def)
	c.Assert(actual, check.Equals, def)

	os.Setenv(cattleVarName, "foobar")
	actual = config.DefaultValue(varName.String(), def)
	c.Assert(actual, check.Equals, "foobar")

	config.SetSecretKey("supersecretkey")
	actual = config.DefaultValue("SECRET_KEY", def)
	c.Assert(actual, check.Equals, "supersecretkey")
}

func setupDeviceOptionsTest(hostConfig *container.HostConfig, instance *model.Instance, mockDevice string) {

	if deviceOptions, ok := utils.GetFieldsIfExist(instance.Data, "fields", "blkioDeviceOptions"); ok {
		blkioWeightDevice := []*blkiodev.WeightDevice{}
		blkioDeviceReadIOps := []*blkiodev.ThrottleDevice{}
		blkioDeviceWriteBps := []*blkiodev.ThrottleDevice{}
		blkioDeviceReadBps := []*blkiodev.ThrottleDevice{}
		blkioDeviceWriteIOps := []*blkiodev.ThrottleDevice{}

		deviceOptions := deviceOptions.(map[string]interface{})
		for dev, options := range deviceOptions {
			if dev == "DEFAULT_DISK" {
				// mock data
				dev = mockDevice
				if dev == "" {
					logrus.Warn(fmt.Sprintf("Couldn't find default device. Not setting device options: %s", options))
					continue
				}
			}
			options := options.(map[string]interface{})
			for key, value := range options {
				value := utils.InterfaceToFloat(value)
				switch key {
				case "weight":
					blkioWeightDevice = append(blkioWeightDevice, &blkiodev.WeightDevice{
						Path:   dev,
						Weight: uint16(value),
					})
					break
				case "readIops":
					blkioDeviceReadIOps = append(blkioDeviceReadIOps, &blkiodev.ThrottleDevice{
						Path: dev,
						Rate: uint64(value),
					})
					break
				case "writeIops":
					blkioDeviceWriteIOps = append(blkioDeviceWriteIOps, &blkiodev.ThrottleDevice{
						Path: dev,
						Rate: uint64(value),
					})
					break
				case "readBps":
					blkioDeviceReadBps = append(blkioDeviceReadBps, &blkiodev.ThrottleDevice{
						Path: dev,
						Rate: uint64(value),
					})
					break
				case "writeBps":
					blkioDeviceWriteBps = append(blkioDeviceWriteBps, &blkiodev.ThrottleDevice{
						Path: dev,
						Rate: uint64(value),
					})
					break
				}
			}
		}
		if len(blkioWeightDevice) > 0 {
			hostConfig.BlkioWeightDevice = blkioWeightDevice
		}
		if len(blkioDeviceReadIOps) > 0 {
			hostConfig.BlkioDeviceReadIOps = blkioDeviceReadIOps
		}
		if len(blkioDeviceWriteIOps) > 0 {
			hostConfig.BlkioDeviceWriteIOps = blkioDeviceWriteIOps
		}
		if len(blkioDeviceReadBps) > 0 {
			hostConfig.BlkioDeviceReadBps = blkioDeviceReadBps
		}
		if len(blkioDeviceWriteBps) > 0 {
			hostConfig.BlkioDeviceWriteBps = blkioDeviceWriteBps
		}
	}
}

func deleteContainer(name string) {
	client := docker.DefaultClient
	containerList, _ := client.ContainerList(context.Background(), types.ContainerListOptions{All: true})
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
			client.ContainerKill(context.Background(), c.ID, "KILL")
			for i := 0; i < 10; i++ {
				if inspect, err := client.ContainerInspect(context.Background(), c.ID); err == nil && inspect.State.Pid == 0 {
					break
				}
				time.Sleep(time.Duration(500) * time.Millisecond)
			}
			client.ContainerRemove(context.Background(), c.ID, types.ContainerRemoveOptions{})
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
