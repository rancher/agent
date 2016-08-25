package hostInfo

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/types"
	"github.com/pkg/errors"
	"github.com/rancher/agent/utilities/config"
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

func (m MockNonIntelCPUInfoGetter) GetCPUInfoData() []string {
	file, err := os.Open("./test_events/cpuinfo")
	defer file.Close()
	data := []string{}
	if err != nil {
		logrus.Error(err)
	} else {
		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			data = append(data, scanner.Text())
		}
	}
	for i, line := range data {
		if strings.HasPrefix(line, "model name") {
			data[i] = "model name : AMD Opteron 250\n"
		}
	}
	return data
}

func (m MockOSCollector) GetOS() map[string]interface{} {
	data := map[string]interface{}{}

	data["operatingSystem"] = "Linux"
	data["kernelVersion"] = "3.19.0-28-generic"

	return data
}

func (m MockOSCollector) GetDockerVersion(verbose bool) map[string]interface{} {
	data := map[string]interface{}{}
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

func (m MockOSCollector) GetWindowsOS() map[string]interface{} {
	return map[string]interface{}{}
}

func (m MockCadvisorGetter) GetContainers() (map[string]interface{}, error) {
	return loadJSON("./test_events/cadvisor_stats")
}

func (m MockCadvisorGetter) GetMachineStats() (map[string]interface{}, error) {
	return loadJSON("./test_events/cadvisor_machine")
}

func (m MockCPUInfoGetter) GetCPULoadAverage() map[string]interface{} {
	return map[string]interface{}{
		"loadAvg": []string{"1.60693359375", "1.73193359375", "1.79248046875"},
	}
}

func (m MockNonIntelCPUInfoGetter) GetCPULoadAverage() map[string]interface{} {
	return map[string]interface{}{
		"loadAvg": []string{"1.60693359375", "1.73193359375", "1.79248046875"},
	}
}

func (m MockCPUInfoGetter) GetCPUInfoData() []string {
	file, err := os.Open("./test_events/cpuinfo")
	defer file.Close()
	data := []string{}
	if err != nil {
		logrus.Error(err)
	} else {
		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			data = append(data, scanner.Text())
		}
	}
	return data
}

func (m MockMemoryInfoGetter) GetMemInfoData() []string {
	file, err := os.Open("./test_events/meminfo")
	defer file.Close()
	data := []string{}
	if err != nil {
		logrus.Error(err)
	} else {
		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			data = append(data, scanner.Text())
		}
	}
	return data
}

func (m MockDiskInfoGetter) GetDockerStorageInfo() map[string]interface{} {
	data := map[string]interface{}{}

	info, err := getMockClientInfo()
	if err != nil {
		logrus.Error(err)
	} else {
		for _, item := range info.DriverStatus {
			data[item[0]] = item[1]
		}
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

func getMockClientInfo() (types.Info, error) {
	info, err := docker.DefaultClient.Info(context.Background())
	if err != nil {
		return info, err
	}
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
	return info, nil
}

var mockCadvisorAPIClient = CadvisorAPIClient{
	dataGetter: MockCadvisorGetter{
		URL: fmt.Sprintf("%v%v:%v/api/%v", "http://", config.CadvisorIP(), config.CadvisorPort(), "v1.2")},
}

var mockCadvisorAPIClientBad = CadvisorAPIClient{
	dataGetter: MockBadCadviosr{},
}

var mockCollectors = []Collector{
	CPUCollector{
		cadvisor:   mockCadvisorAPIClient,
		dataGetter: MockCPUInfoGetter{},
		GOOS:       "linux",
	},
	DiskCollector{
		cadvisor:            mockCadvisorAPIClient,
		dockerStorageDriver: "devicemapper",
		unit:                1048576,
		dataGetter:          MockDiskInfoGetter{},
	},
	IopsCollector{
		GOOS: "linux",
	},
	MemoryCollector{
		unit:       1024.00,
		dataGetter: MockMemoryInfoGetter{},
		GOOS:       "linux",
	},
	OSCollector{
		dataGetter: MockOSCollector{},
		GOOS:       "linux",
	},
}

var mockCollectorsBadCadvisor = []Collector{
	CPUCollector{
		cadvisor:   mockCadvisorAPIClientBad,
		dataGetter: MockCPUInfoGetter{},
		GOOS:       "linux",
	},
	DiskCollector{
		cadvisor:            mockCadvisorAPIClientBad,
		dockerStorageDriver: "devicemapper",
		unit:                1048576,
		dataGetter:          MockDiskInfoGetter{},
	},
	IopsCollector{
		GOOS: "linux",
	},
	MemoryCollector{
		unit:       1024.00,
		dataGetter: MockMemoryInfoGetter{},
		GOOS:       "linux",
	},
	OSCollector{
		dataGetter: MockOSCollector{},
		GOOS:       "linux",
	},
}

var mockNonCadvisorNonIntelCPUInfoMock = []Collector{
	CPUCollector{
		cadvisor:   mockCadvisorAPIClientBad,
		dataGetter: MockNonIntelCPUInfoGetter{},
		GOOS:       "linux",
	},
	DiskCollector{
		cadvisor:            mockCadvisorAPIClientBad,
		dockerStorageDriver: "devicemapper",
		unit:                1048576,
		dataGetter:          DiskDataGetter{},
	},
	IopsCollector{
		GOOS: "linux",
	},
	MemoryCollector{
		unit:       1024.00,
		dataGetter: MockMemoryInfoGetter{},
		GOOS:       "linux",
	},
	OSCollector{
		dataGetter: MockOSCollector{},
		GOOS:       "linux",
	},
}

var mockNonLinux = []Collector{
	CPUCollector{
		cadvisor:   Cadvisor,
		dataGetter: CPUDataGetter{},
		GOOS:       "nonlinux",
	},
	DiskCollector{
		cadvisor:            Cadvisor,
		dockerStorageDriver: utils.GetInfoDriver(),
		unit:                1048576,
		dataGetter:          DiskDataGetter{},
	},
	IopsCollector{
		GOOS: "nonlinux",
	},
	MemoryCollector{
		unit:       1024.00,
		dataGetter: MemoryDataGetter{},
		GOOS:       "nonlinux",
	},
	OSCollector{
		dataGetter: OSDataGetter{},
		GOOS:       "nonlinux",
	},
}

func MockHostLabels(prefix string) map[string]string {
	labels := map[string]string{}
	for _, collector := range mockCollectors {
		for key, value := range collector.GetLabels(prefix) {
			labels[key] = value
		}
	}
	return labels
}

func MockCollectData(number int) map[string]interface{} {
	data := map[string]interface{}{}
	switch number {
	case 1:
		for _, collector := range mockCollectorsBadCadvisor {
			data[collector.KeyName()] = collector.GetData()
		}
	case 2:
		for _, collector := range mockCollectors {
			data[collector.KeyName()] = collector.GetData()
		}
	case 3:
		for _, collector := range mockNonCadvisorNonIntelCPUInfoMock {
			data[collector.KeyName()] = collector.GetData()
		}
	case 4:
		for _, collector := range mockNonLinux {
			data[collector.KeyName()] = collector.GetData()
		}
	}
	return data
}

var Labels = MockHostLabels("io.rancher.host")

var HostData = MockCollectData(2)

var BadHostData = MockCollectData(1)

var NonIntelHostData = MockCollectData(3)

var NonLinuxHostData = MockCollectData(4)

func (s *ComputeTestSuite) TestHostLabel(c *check.C) {
	expected := map[string]string{
		"io.rancher.host.docker_version":       "1.6",
		"io.rancher.host.linux_kernel_version": "3.19",
	}
	hostLabels := MockHostLabels("io.rancher.host")
	delete(hostLabels, "io.rancher.host.kvm")
	c.Assert(hostLabels, check.DeepEquals, expected)
}

func (s *ComputeTestSuite) TestCadvisorTime(c *check.C) {
	cadvisorClient := CadvisorAPIClient{
		dataGetter: CadvisorDataGetter{
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
	for key := range utils.InterfaceToMap(HostData["osInfo"]) {
		obtainedKeys = append(obtainedKeys, key)
	}
	sort.Strings(expectKeys)
	sort.Strings(obtainedKeys)
	c.Assert(obtainedKeys, check.DeepEquals, expectKeys)

	version, err := utils.GetFieldsIfExist(HostData, "osInfo", "dockerVersion")
	if !err {
		c.Fatal("No dockerVersion found")
	}
	c.Assert(utils.InterfaceToString(version), check.Equals,
		"Docker version 1.6.0, build 4749651")
	operatingSystem, err := utils.GetFieldsIfExist(HostData, "osInfo", "operatingSystem")
	if !err {
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
	mhz, ok := utils.GetFieldsIfExist(NonIntelHostData, "cpuInfo", "mhz")
	if !ok {
		c.Fatal("No mhz found")
	}
	mhz = utils.InterfaceToFloat(mhz)
	c.Assert(mhz, check.Equals, 2334.915)
}

func (s *ComputeTestSuite) TestNonLinuxHost(c *check.C) {
	expectKeys := []string{
		"memoryInfo",
		"osInfo",
		"cpuInfo",
		"diskInfo",
		"iopsInfo",
	}
	obtainedKeys := []string{}
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
