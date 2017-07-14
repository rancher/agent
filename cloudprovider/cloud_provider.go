package cloudprovider

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/rancher/agent/cloudprovider/aliyun"
	"github.com/rancher/agent/cloudprovider/aws"
)

const (
	AwsEndpoint    = "http://169.254.169.254"
	AliyunEndpoint = "http://100.100.100.200"
)

var allProviderName = []string{"aws", "aliyun"}

func GetCloudProviderInfo() {
	if len(allProviderName) <= 0 {
		return
	}
	for _, providerName := range allProviderName {
		switch providerName {
		case "aws":
			AwsProvider(providerName)
		case "aliyun":
			AliyunProvider(providerName)
		}
	}
}

func AwsProvider(providerName string) {
	if providerName != "aws" {
		return
	}
	for i := 0; i < 3; i++ {
		err := PingServer(providerName)
		if err == nil {
			break
		} else if err != nil && i == 2 {
			return
		}
	}
	provider := aws.NewProvider()
	go provider.GetCloudProviderInfo()
}

func AliyunProvider(providerName string) {
	if providerName != "aliyun" {
		return
	}
	for i := 0; i < 3; i++ {
		err := PingServer(providerName)
		if err == nil {
			break
		} else if err != nil && i == 2 {
			return
		}
	}
	provider := aliyun.NewProvider()
	go provider.GetCloudProviderInfo()
}

func PingServer(server string) error {
	var url string
	client := &http.Client{}
	switch server {
	case "aws":
		url = AwsEndpoint + "/latest"
	case "aliyun":
		url = AliyunEndpoint + "/latest"
	default:
		return fmt.Errorf("unknow cloud provider")
	}
	requ, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(requ)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("server unconnected")
	}
	defer resp.Body.Close()
	metaData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("get metadata err")
	}
	if !strings.Contains(string(metaData), "meta-data") {
		return fmt.Errorf("unknow respose")
	}
	return nil

}
