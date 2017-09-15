package util

import (
	"github.com/rancher/agent/service/hostapi/config"
	rclient "github.com/rancher/go-rancher/v3"
)

func GetRancherClient() (*rclient.RancherClient, error) {
	apiURL := config.Config.CattleURL
	accessKey := config.Config.CattleAccessKey
	secretKey := config.Config.CattleSecretKey

	if apiURL == "" || accessKey == "" || secretKey == "" {
		return nil, nil
	}

	apiClient, err := rclient.NewRancherClient(&rclient.ClientOpts{
		Url:       apiURL,
		AccessKey: accessKey,
		SecretKey: secretKey,
	})
	if err != nil {
		return nil, err
	}
	return apiClient, nil
}
