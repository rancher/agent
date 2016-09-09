package hostInfo

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/docker/engine-api/types"
	"github.com/pkg/errors"
	"github.com/rancher/agent/model"
	"github.com/rancher/agent/utilities/config"
	"github.com/rancher/agent/utilities/constants"
	"github.com/rancher/agent/utilities/docker"
	"github.com/rancher/agent/utilities/utils"
	"golang.org/x/net/context"
	"gopkg.in/check.v1"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"testing"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	check.TestingT(t)
}

type ComputeTestSuite struct{}

var _ = check.Suite(&ComputeTestSuite{})

func (s *ComputeTestSuite) SetUpSuite(c *check.C) {}

type MockCPUInfoGetter struct{}

type MockMemoryInfoGetter struct{}

type MockCadvisorGetter struct {
	URL string
}

type MockDiskInfoGetter struct{}

type MockOSCollector struct{}

type MockBadCadviosr struct{}

type MockNonIntelCPUInfoGetter struct{}

func (m MockNonIntelCPUInfoGetter) GetCPUInfoData() ([]string, error) {
	file, err := os.Open("./test_events/cpuinfo")
	defer file.Close()
	data := []string{}
	if err != nil {
		return data, err
	}
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		data = append(data, scanner.Text())
	}
	for i, line := range data {
		if strings.HasPrefix(line, "model name") {
			data[i] = "model name : AMD Opteron 250\n"
		}
	}
	return data, nil
}

func (m MockOSCollector) GetOS(infoData model.InfoData) (map[string]string, error) {
	data := map[string]string{}

	data["operatingSystem"] = "Linux"
	data["kernelVersion"] = "3.19.0-28-generic"

	return data, nil
}

func (m MockOSCollector) GetDockerVersion(infoData model.InfoData, verbose bool) map[string]string {
	data := map[string]string{}
	verResp := types.Version{
		KernelVersion: "4.0.3-boot2docker",
		Arch:          "amd64",
		APIVersion:    "1.18",
		Version:       "1.6.0",
		GitCommit:     "4749651",
		Os:            "linux",
		GoVersion:     "go1.4.2",
	}

	version := "unknown"
	if verbose && verResp.Version != "" {
		version = fmt.Sprintf("Docker version %v, build %v", verResp.Version, verResp.GitCommit)
	} else if verResp.Version != "" {
		version = utils.SemverTrunk(verResp.Version, 2)
	}
	data["dockerVersion"] = version

	return data
}

func (m MockCadvisorGetter) GetContainers() (map[string]interface{}, error) {
	return loadJSON("./test_events/cadvisor_stats")
}

func (m MockCadvisorGetter) GetMachineStats() (map[string]interface{}, error) {
	return loadJSON("./test_events/cadvisor_machine")
}

func (m MockCPUInfoGetter) GetCPULoadAverage() (map[string]interface{}, error) {
	return map[string]interface{}{
		"loadAvg": []string{"1.60693359375", "1.73193359375", "1.79248046875"},
	}, nil
}

func (m MockNonIntelCPUInfoGetter) GetCPULoadAverage() (map[string]interface{}, error) {
	return map[string]interface{}{
		"loadAvg": []string{"1.60693359375", "1.73193359375", "1.79248046875"},
	}, nil
}

func (m MockCPUInfoGetter) GetCPUInfoData() ([]string, error) {
	file, err := os.Open("./test_events/cpuinfo")
	defer file.Close()
	data := []string{}
	if err != nil {
		return []string{}, err
	}
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		data = append(data, scanner.Text())
	}

	return data, nil
}

func (m MockMemoryInfoGetter) GetMemInfoData() ([]string, error) {
	file, err := os.Open("./test_events/meminfo")
	defer file.Close()
	data := []string{}
	if err != nil {
		return data, err
	}
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		data = append(data, scanner.Text())
	}
	return data, nil
}

func (m MockDiskInfoGetter) GetDockerStorageInfo(infoData model.InfoData) map[string]interface{} {
	data := map[string]interface{}{}

	info := getMockClientInfo()
	for _, item := range info.DriverStatus {
		data[item[0]] = item[1]
	}

	return data
}

func (m MockCadvisorGetter) Get(url string) (map[string]interface{}, error) {
	return utils.Get(url)
}

func (m MockBadCadviosr) GetContainers() (map[string]interface{}, error) {
	return loadJSON("./test_events/cadvisor_stats")
}

func (m MockBadCadviosr) GetMachineStats() (map[string]interface{}, error) {
	return loadJSON("./test_events/cadvisor_machine")
}

func (m MockBadCadviosr) Get(url string) (map[string]interface{}, error) {
	return nil, errors.New("BAD ERROR")
}

func getMockClientInfo() types.Info {
	info, _ := docker.GetClient(constants.DefaultVersion).Info(context.Background())
	info.Driver = "devicemapper"
	info.DriverStatus = [][2]string{
		[2]string{"Pool Name", "docker-8:1-130861-pool"},
		[2]string{"Pool Blocksize", "65.54 kB"},
		[2]string{"Backing Filesystem", "extfs"},
		[2]string{"Data file", "/dev/loop0"},
		[2]string{"Metadata file", "/dev/loop1"},
		[2]string{"Data Space Used", "2.661 GB"},
		[2]string{"Data Space Total", "107.4 GB"},
		[2]string{"Data Space Available", "16.8 GB"},
		[2]string{"Metadata Space Used", "2.683 MB"},
		[2]string{"Metadata Space Total", "2.147 GB"},
		[2]string{"Metadata Space Available", "2.145 GB"},
		[2]string{"Udev Sync Supported", "false"},
		[2]string{"Deferred Removal Enabled", "false"},
		[2]string{"Data loop file", "/mnt/sda1/var/lib/docker/devicemapper/devicemapper/data"},
		[2]string{"Metadata loop file", "/mnt/sda1/var/lib/docker/devicemapper/devicemapper/metadata"},
		[2]string{"Library Version", "1.02.82-git (2013-10-04)"},
	}
	return info
}

var mockCadvisorAPIClient = CadvisorAPIClient{
	DataGetter: MockCadvisorGetter{
		URL: fmt.Sprintf("%v%v:%v/api/%v", "http://", config.CadvisorIP(), config.CadvisorPort(), "v1.2")},
}

var mockCadvisorAPIClientBad = CadvisorAPIClient{
	DataGetter: MockBadCadviosr{},
}

var mockCollectors = []Collector{
	CPUCollector{
		Cadvisor:   mockCadvisorAPIClient,
		DataGetter: MockCPUInfoGetter{},
		GOOS:       "linux",
	},
	DiskCollector{
		Cadvisor:            mockCadvisorAPIClient,
		DockerStorageDriver: "devicemapper",
		Unit:                1048576,
		DataGetter:          MockDiskInfoGetter{},
	},
	IopsCollector{
		GOOS: "linux",
	},
	MemoryCollector{
		Unit:       1024.00,
		DataGetter: MockMemoryInfoGetter{},
		GOOS:       "linux",
	},
	OSCollector{
		DataGetter: MockOSCollector{},
		GOOS:       "linux",
	},
}

var mockCollectorsBadCadvisor = []Collector{
	CPUCollector{
		Cadvisor:   mockCadvisorAPIClientBad,
		DataGetter: MockCPUInfoGetter{},
		GOOS:       "linux",
	},
	DiskCollector{
		Cadvisor:            mockCadvisorAPIClientBad,
		DockerStorageDriver: "devicemapper",
		Unit:                1048576,
		DataGetter:          MockDiskInfoGetter{},
	},
	IopsCollector{
		GOOS: "linux",
	},
	MemoryCollector{
		Unit:       1024.00,
		DataGetter: MockMemoryInfoGetter{},
		GOOS:       "linux",
	},
	OSCollector{
		DataGetter: MockOSCollector{},
		GOOS:       "linux",
	},
}

var mockNonCadvisorNonIntelCPUInfoMock = []Collector{
	CPUCollector{
		Cadvisor:   mockCadvisorAPIClientBad,
		DataGetter: MockNonIntelCPUInfoGetter{},
		GOOS:       "linux",
	},
	DiskCollector{
		Cadvisor:            mockCadvisorAPIClientBad,
		DockerStorageDriver: "devicemapper",
		Unit:                1048576,
		DataGetter:          DiskDataGetter{},
	},
	IopsCollector{
		GOOS: "linux",
	},
	MemoryCollector{
		Unit:       1024.00,
		DataGetter: MockMemoryInfoGetter{},
		GOOS:       "linux",
	},
	OSCollector{
		DataGetter: MockOSCollector{},
		GOOS:       "linux",
	},
}
var Cadvisor = CadvisorAPIClient{
	DataGetter: CadvisorDataGetter{
		URL: fmt.Sprintf("%v%v:%v/api/%v", "http://", config.CadvisorIP(), config.CadvisorPort(), "v1.2"),
	},
}

var mockNonLinux = []Collector{
	CPUCollector{
		Cadvisor:   Cadvisor,
		DataGetter: CPUDataGetter{},
		GOOS:       "nonlinux",
	},
	DiskCollector{
		Cadvisor:   Cadvisor,
		Unit:       1048576,
		DataGetter: DiskDataGetter{},
	},
	IopsCollector{
		GOOS: "nonlinux",
	},
	MemoryCollector{
		Unit:       1024.00,
		DataGetter: MemoryDataGetter{},
		GOOS:       "nonlinux",
	},
	OSCollector{
		DataGetter: OSDataGetter{},
		GOOS:       "nonlinux",
	},
}

func MockHostLabels(prefix string) (map[string]string, error) {
	labels := map[string]string{}
	for _, collector := range mockCollectors {
		ls, err := collector.GetLabels(prefix)
		if err != nil {
			return map[string]string{}, err
		}
		for key, value := range ls {
			labels[key] = value
		}
	}
	return labels, nil
}

func MockCollectData(name string) (map[string]interface{}, error) {
	data := map[string]interface{}{}
	switch name {
	case "BadCadvisor":
		for _, collector := range mockCollectorsBadCadvisor {
			collectData, err := collector.GetData()
			if err != nil {
				return map[string]interface{}{}, err
			}
			data[collector.KeyName()] = collectData
		}
	case "Mock":
		for _, collector := range mockCollectors {
			collectData, err := collector.GetData()
			if err != nil {
				return map[string]interface{}{}, err
			}
			data[collector.KeyName()] = collectData
		}
	case "NonIntelCPU":
		for _, collector := range mockNonCadvisorNonIntelCPUInfoMock {
			collectData, err := collector.GetData()
			if err != nil {
				return map[string]interface{}{}, err
			}
			data[collector.KeyName()] = collectData
		}
	case "NonLinux":
		for _, collector := range mockNonLinux {
			collectData, err := collector.GetData()
			if err != nil {
				return map[string]interface{}{}, err
			}
			data[collector.KeyName()] = collectData
		}
	}
	return data, nil
}

//var Labels = MockHostLabels("io.rancher.host")

//var HostData = MockCollectData(2)
//
//var BadHostData = MockCollectData(1)
//
//var NonIntelHostData = MockCollectData(3)
//
//var NonLinuxHostData = MockCollectData(4)

func (s *ComputeTestSuite) TestHostLabel(c *check.C) {
	expected := map[string]string{
		"io.rancher.host.docker_version":       "1.6",
		"io.rancher.host.linux_kernel_version": "3.19",
	}
	hostLabels, err := MockHostLabels("io.rancher.host")
	if err != nil {
		c.Fatal(err)
	}
	delete(hostLabels, "io.rancher.host.kvm")
	c.Assert(hostLabels, check.DeepEquals, expected)
}

func (s *ComputeTestSuite) TestCadvisorTime(c *check.C) {
	cadvisorClient := CadvisorAPIClient{
		DataGetter: CadvisorDataGetter{
			URL: fmt.Sprintf("%v%v:%v/api/%v", "http://", config.CadvisorIP(), config.CadvisorPort(), "v1.2"),
		},
	}
	timeVals := map[string]string{
		"2015-02-04T23:21:38.251266323-07:00": "2015-01-28T20:08:16.967019892Z",
		"2015-02-04T23:21:38.251266323+07:00": "2015-01-28T20:08:16.967019892Z",
	}
	for key, value := range timeVals {
		val := cadvisorClient.TimestampDiff(key, value)
		c.Assert(val, check.FitsTypeOf, float64(0.0))
	}
}

func (s *ComputeTestSuite) TestCollectDataMeminfo(c *check.C) {
	expectMemKeys := []string{
		"memTotal",
		"memFree",
		"memAvailable",
		"buffers",
		"cached",
		"swapCached",
		"active",
		"inactive",
		"swapTotal",
		"swapFree",
	}
	obtainedKeys := []string{}
	HostData, err := MockCollectData("Mock")
	if err != nil {
		c.Fatal(err)
	}
	for key := range utils.InterfaceToMap(HostData["memoryInfo"]) {
		obtainedKeys = append(obtainedKeys, key)
	}
	sort.Strings(obtainedKeys)
	sort.Strings(expectMemKeys)
	c.Assert(obtainedKeys, check.DeepEquals, expectMemKeys)
}

func (s *ComputeTestSuite) TestCollectDataOSInfo(c *check.C) {
	expectKeys := []string{
		"operatingSystem",
		"dockerVersion",
		"kernelVersion",
	}
	obtainedKeys := []string{}
	HostData, err := MockCollectData("Mock")
	if err != nil {
		c.Fatal(err)
	}
	for key := range utils.InterfaceToMap(HostData["osInfo"]) {
		obtainedKeys = append(obtainedKeys, key)
	}
	sort.Strings(expectKeys)
	sort.Strings(obtainedKeys)
	c.Assert(obtainedKeys, check.DeepEquals, expectKeys)

	version, ok := utils.GetFieldsIfExist(HostData, "osInfo", "dockerVersion")
	if !ok {
		c.Fatal("No dockerVersion found")
	}
	c.Assert(utils.InterfaceToString(version), check.Equals,
		"Docker version 1.6.0, build 4749651")
	operatingSystem, ok := utils.GetFieldsIfExist(HostData, "osInfo", "operatingSystem")
	if !ok {
		c.Fatal("No os found")
	}
	c.Assert(utils.InterfaceToString(operatingSystem), check.Equals, "Linux")
}

func (s *ComputeTestSuite) TestCollectDataDiskInfo(c *check.C) {
	expectKeys := []string{
		"fileSystems",
		"mountPoints",
		"dockerStorageDriver",
		"dockerStorageDriverStatus",
	}
	obtainedKeys := []string{}
	HostData, err := MockCollectData("Mock")
	if err != nil {
		c.Fatal(err)
	}
	for key := range utils.InterfaceToMap(HostData["diskInfo"]) {
		obtainedKeys = append(obtainedKeys, key)
	}
	sort.Strings(obtainedKeys)
	sort.Strings(expectKeys)
	c.Assert(obtainedKeys, check.DeepEquals, expectKeys)
	mountsPoint, ok := utils.GetFieldsIfExist(HostData, "diskInfo", "mountPoints")
	if !ok {
		c.Fatal("No MountPoints found")
	}
	obtainedMountKeys := []string{}
	for key := range utils.InterfaceToMap(mountsPoint) {
		obtainedMountKeys = append(obtainedMountKeys, key)
	}
	c.Assert(obtainedMountKeys, check.DeepEquals, []string{"/dev/sda1"})
	filesystem, ok := utils.GetFieldsIfExist(HostData, "diskInfo", "fileSystems")
	if !ok {
		c.Fatal("No fileSystems found")
	}
	obtainedFileSystemsKeys := []string{}
	for key := range utils.InterfaceToMap(filesystem) {
		obtainedFileSystemsKeys = append(obtainedFileSystemsKeys, key)
	}
	c.Assert(len(obtainedFileSystemsKeys), check.Not(check.Equals), 0)
	_, ok = utils.InterfaceToMap(filesystem)["/dev/mapper/docker-8:1-130861-c3ae1852921c3fec9c9a74dce987f47f7e1ae8e7e3bcd9ad98e671f5d80a28d8"]
	c.Assert(ok, check.Equals, false)
}

func (s *ComputeTestSuite) TestCollectDataBadCadvisorStat(c *check.C) {
	expectedCPUKeys := []string{
		"modelName",
		"count",
		"mhz",
		"loadAvg",
		"cpuCoresPercentages",
	}
	obtainedCPUKeys := []string{}
	BadHostData, err := MockCollectData("BadCadvisor")
	if err != nil {
		c.Fatal(err)
	}
	for key := range utils.InterfaceToMap(BadHostData["cpuInfo"]) {
		obtainedCPUKeys = append(obtainedCPUKeys, key)
	}
	sort.Strings(expectedCPUKeys)
	sort.Strings(obtainedCPUKeys)
	c.Assert(obtainedCPUKeys, check.DeepEquals, expectedCPUKeys)
	cpuPercentage, ok := utils.GetFieldsIfExist(BadHostData, "cpuInfo", "cpuCoresPercentages")
	if !ok {
		c.Fatal("No cpuCoresPercentages found")
	}
	c.Assert(len(utils.InterfaceToArray(cpuPercentage)), check.Equals, 0)
	expectedDiskKeys := []string{
		"fileSystems",
		"mountPoints",
		"dockerStorageDriver",
		"dockerStorageDriverStatus",
	}
	obtainedDiskKeys := []string{}
	for key := range utils.InterfaceToMap(BadHostData["diskInfo"]) {
		obtainedDiskKeys = append(obtainedDiskKeys, key)
	}
	sort.Strings(expectedDiskKeys)
	sort.Strings(obtainedDiskKeys)

	c.Assert(obtainedCPUKeys, check.DeepEquals, expectedCPUKeys)
}

func (s *ComputeTestSuite) TestCollectDataCPUInfo(c *check.C) {
	expectedKeys := []string{
		"modelName",
		"count",
		"mhz",
		"loadAvg",
		"cpuCoresPercentages",
	}
	obtainedKeys := []string{}
	HostData, err := MockCollectData("Mock")
	if err != nil {
		c.Fatal(err)
	}
	for key := range utils.InterfaceToMap(HostData["cpuInfo"]) {
		obtainedKeys = append(obtainedKeys, key)
	}
	sort.Strings(obtainedKeys)
	sort.Strings(expectedKeys)
	c.Assert(obtainedKeys, check.DeepEquals, expectedKeys)
	modelName, ok := utils.GetFieldsIfExist(HostData, "cpuInfo", "modelName")
	if !ok {
		c.Fatal("No modelName found")
	}
	modelName = utils.InterfaceToString(modelName)
	c.Assert(modelName, check.Equals, "Intel(R) Core(TM) i7-4650U CPU @ 1.70GHz")
	mhz, ok := utils.GetFieldsIfExist(HostData, "cpuInfo", "mhz")
	if !ok {
		c.Fatal("No mhz found")
	}
	mhz = utils.InterfaceToFloat(mhz)
	c.Assert(mhz, check.Equals, 1700.0)
}

func (s *ComputeTestSuite) TestCollectDataCPUFreqFallBack(c *check.C) {
	NonIntelHostData, err := MockCollectData("NonIntelCPU")
	if err != nil {
		c.Fatal(err)
	}
	mhz, ok := utils.GetFieldsIfExist(NonIntelHostData, "cpuInfo", "mhz")
	if !ok {
		c.Fatal("No mhz found")
	}
	mhz = utils.InterfaceToFloat(mhz)
	c.Assert(mhz, check.Equals, 2334.915)
}

// remove this test bc Non linux host like windows should also return host data
/*
func (s *ComputeTestSuite) TestNonLinuxHost(c *check.C) {
	expectKeys := []string{
		"memoryInfo",
		"osInfo",
		"cpuInfo",
		"diskInfo",
		"iopsInfo",
	}
	obtainedKeys := []string{}
	NonLinuxHostData, err := MockCollectData("NonLinux")
	if err != nil {
		c.Fatal(err)
	}
	for key := range utils.InterfaceToMap(NonLinuxHostData) {
		obtainedKeys = append(obtainedKeys, key)
	}
	sort.Strings(expectKeys)
	sort.Strings(obtainedKeys)
	c.Assert(obtainedKeys, check.DeepEquals, expectKeys)
	memoryInfo := utils.InterfaceToMap(NonLinuxHostData["memoryInfo"])
	osInfo := utils.InterfaceToMap(NonLinuxHostData["osInfo"])
	cpuInfo := utils.InterfaceToMap(NonLinuxHostData["cpuInfo"])
	c.Assert(len(memoryInfo), check.Equals, 0)
	c.Assert(len(osInfo), check.Equals, 1)
	c.Assert(len(cpuInfo), check.Equals, 0)
	_, ok1 := utils.InterfaceToMap(NonLinuxHostData["diskInfo"])["mountPoints"]
	_, ok2 := utils.InterfaceToMap(NonLinuxHostData["diskInfo"])["fileSystems"]
	_, ok3 := utils.InterfaceToMap(NonLinuxHostData["diskInfo"])["dockerStorageDriver"]
	_, ok4 := utils.InterfaceToMap(NonLinuxHostData["diskInfo"])["dockerStorageDriverStatus"]
	c.Assert(ok1, check.Equals, true)
	c.Assert(ok2, check.Equals, true)
	c.Assert(ok3, check.Equals, true)
	c.Assert(ok4, check.Equals, true)
}
*/

func loadJSON(path string) (map[string]interface{}, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	event := map[string]interface{}{}
	err = json.Unmarshal(file, &event)
	if err != nil {
		return nil, err
	}
	return event, nil
}
