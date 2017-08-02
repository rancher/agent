package hostapi

import (
	"os"
	"time"

	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/golang/glog"
	"github.com/rancher/agent/service/hostapi/config"
	"github.com/rancher/agent/service/hostapi/console"
	"github.com/rancher/agent/service/hostapi/dockersocketproxy"
	"github.com/rancher/agent/service/hostapi/events"
	"github.com/rancher/agent/service/hostapi/exec"
	"github.com/rancher/agent/service/hostapi/logs"
	"github.com/rancher/agent/service/hostapi/proxy"
	"github.com/rancher/agent/service/hostapi/stats"
	"github.com/rancher/agent/service/hostapi/util"
	rclient "github.com/rancher/go-rancher/client"
	"github.com/rancher/websocket-proxy/backend"
	"io/ioutil"
	"strconv"
	"strings"
)

func StartUp() {
	for {
		err := config.Parse()
		if err != nil {
			time.Sleep(time.Duration(5) * time.Second)
			continue
		}
		defer glog.Flush()
		if config.Config.PidFile != "" {
			logrus.Infof("Writing pid %d to %s", os.Getpid(), config.Config.PidFile)
			if err := ioutil.WriteFile(config.Config.PidFile, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
				logrus.Errorf("Failed to write pid file %s: %v", config.Config.PidFile, err)
				time.Sleep(time.Duration(5) * time.Second)
				continue
			}
		}
		if config.Config.LogFile != "" {
			if output, err := os.OpenFile(config.Config.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666); err != nil {
				logrus.Errorf("Failed to log to file %s: %v", config.Config.LogFile, err)
				time.Sleep(time.Duration(5) * time.Second)
				continue
			} else {
				logrus.SetOutput(output)
			}
		}
		processor := events.NewDockerEventsProcessor(config.Config.EventsPoolSize)
		err = processor.Process()
		if err != nil {
			logrus.Error(err)
			time.Sleep(time.Duration(5) * time.Second)
			continue
		}
		break
	}
	for {
		rancherClient, err := util.GetRancherClient()
		if err != nil {
			logrus.Error(err)
			time.Sleep(time.Duration(5) * time.Second)
			continue
		}
		tokenRequest := &rclient.HostApiProxyToken{
			ReportedUuid: config.Config.HostUUID,
		}
		tokenResponse, err := getConnectionToken(0, tokenRequest, rancherClient)
		if err != nil {
			logrus.Error(err)
			time.Sleep(time.Duration(5) * time.Second)
			continue
		} else if tokenResponse == nil {
			// nil error and blank token means the proxy is turned off. Just block forever so main function doesn't exit
			var block chan bool
			<-block
		}

		config.Config.HostUUID = tokenResponse.ReportedUuid
		logrus.Errorf("HostUUID: %s", config.Config.HostUUID)

		handlers := make(map[string]backend.Handler)
		handlers["/v1/logs/"] = &logs.Handler{}
		handlers["/v2-beta/logs/"] = &logs.Handler{}
		handlers["/v1/stats/"] = &stats.Handler{}
		handlers["/v2-beta/stats/"] = &stats.Handler{}
		handlers["/v1/hoststats/"] = &stats.HostStatsHandler{}
		handlers["/v2-beta/hoststats/"] = &stats.HostStatsHandler{}
		handlers["/v1/containerstats/"] = &stats.ContainerStatsHandler{}
		handlers["/v2-beta/containerstats/"] = &stats.ContainerStatsHandler{}
		handlers["/v1/exec/"] = &exec.Handler{}
		handlers["/v2-beta/exec/"] = &exec.Handler{}
		handlers["/v1/console/"] = &console.Handler{}
		handlers["/v2-beta/console/"] = &console.Handler{}
		handlers["/v1/dockersocket/"] = &dockersocketproxy.Handler{}
		handlers["/v2-beta/dockersocket/"] = &dockersocketproxy.Handler{}
		handlers["/v1/container-proxy/"] = &proxy.Handler{}
		handlers["/v2-beta/container-proxy/"] = &proxy.Handler{}
		if err := backend.ConnectToProxy(tokenResponse.Url+"?token="+tokenResponse.Token, handlers); err != nil {
			logrus.Error(err)
			time.Sleep(time.Duration(5) * time.Second)
			continue
		}
	}
}

const maxWaitOnHostTries = 20

func getConnectionToken(try int, tokenReq *rclient.HostApiProxyToken, rancherClient *rclient.RancherClient) (*rclient.HostApiProxyToken, error) {
	if try >= maxWaitOnHostTries {
		return nil, fmt.Errorf("Reached max retry attempts for getting token")
	}

	tokenResponse, err := rancherClient.HostApiProxyToken.Create(tokenReq)
	if err != nil {
		if apiError, ok := err.(*rclient.ApiError); ok {
			if apiError.StatusCode == 422 {
				parsed := &ParsedError{}
				if uErr := json.Unmarshal([]byte(apiError.Body), &parsed); uErr == nil {
					if strings.EqualFold(parsed.Code, "InvalidReference") && strings.EqualFold(parsed.FieldName, "reportedUuid") {
						logrus.WithField("reportedUuid", config.Config.HostUUID).WithField("Attempt", try).Infof("Host not registered yet. Sleeping 1 second and trying again.")
						time.Sleep(time.Second)
						try++
						return getConnectionToken(try, tokenReq, rancherClient) // Recursion!
					}
				} else {
					return nil, uErr
				}
			} else if apiError.StatusCode == 501 {
				logrus.Infof("Host-api proxy disabled. Will not connect.")
				return nil, nil
			}
			return nil, err
		}
	}
	return tokenResponse, nil
}

type ParsedError struct {
	Code      string
	FieldName string
}
