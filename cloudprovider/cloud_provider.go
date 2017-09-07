package cloudprovider

import (
	"github.com/Sirupsen/logrus"
	"github.com/rancher/agent/cloudprovider/aliyun"
	"github.com/rancher/agent/cloudprovider/aws"
)

type Provider interface {
	GetCloudProviderInfo() bool
}

func GetCloudProviderInfo() {
	logrus.Info("Getting aws provider info")
	providerAws := aws.NewProvider()
	go providerAws.GetCloudProviderInfo()

	logrus.Info("Getting aliyun provider info")
	providerAliyun := aliyun.NewProvider()
	go providerAliyun.GetCloudProviderInfo()
}
