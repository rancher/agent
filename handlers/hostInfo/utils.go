package hostInfo

import (
	"bufio"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/types"
	"github.com/rancher/agent/handlers/docker"
	"golang.org/x/net/context"
	"os"
	"regexp"
	"runtime"
	"strings"
)

const DefaultVersion = "1.22"

var ConfigOverride = make(map[string]string)

func semverTrunk(version string, vals int) string {
	/*
			vrm_vals: is a number representing the number of
		        digits to return. ex: 1.8.3
		          vmr_val = 1; return val 1
		          vmr_val = 2; return val 1.8
		          vmr_val = 3; return val 1.8.3
	*/
	if version != "" {
		m := map[int]string{
			1: regexp.MustCompile("(\\d+)").FindString(version),
			2: regexp.MustCompile("(\\d+\\.)?(\\d+)").FindString(version),
			3: regexp.MustCompile("(\\d+\\.)?(\\d+\\.)?(\\d+)").FindString(version),
		}
		return m[vals]
	}
	return version
}

func getKernelVersion() string {
	if runtime.GOOS == "linux" {
		file, err := os.Open("/proc/version")
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
		version := regexp.MustCompile("\\d+.\\d+.\\d+").FindString(data[0])
		return version
	}
	return ""
}

func getLoadAverage() []string {
	if runtime.GOOS == "linux" {
		file, err := os.Open("/proc/loadavg")
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
		loads := strings.Split(data[0], " ")
		return loads[:3]
	}
	return []string{}
}

func getInfo() types.Info {
	info, _ := docker.GetClient(DefaultVersion).Info(context.Background())
	return info
}

func getFieldsIfExist(m map[string]interface{}, fields ...string) (interface{}, bool) {
	var tempMap map[string]interface{}
	tempMap = m
	for i, field := range fields {
		switch tempMap[field].(type) {
		case map[string]interface{}:
			tempMap = tempMap[field].(map[string]interface{})
		case nil:
			return nil, false
		default:
			// if it is the last field and it is not empty
			// it exists othewise return false
			if i == len(fields)-1 {
				return tempMap[field], true
			}
			return nil, false
		}
	}
	return tempMap, true
}

func CadvisorPort() string {
	return defaultValue("CADVISOR", "9344")
}

func CadvisorIP() string {
	return defaultValue("CADVISOR", "127.0.0.1")
}

func defaultValue(name string, df string) string {
	if value, ok := ConfigOverride[name]; ok {
		return value
	}
	if result := os.Getenv(fmt.Sprintf("CATTLE_%s", name)); result != "" {
		return result
	}
	return df
}
