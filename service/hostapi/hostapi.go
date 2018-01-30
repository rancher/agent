package hostapi

import (
	"os"
	"time"

	"io/ioutil"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/golang/glog"
	"github.com/pkg/errors"
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
				logrus.Errorf("Failed to write pid file [%s] for host-api startup: %v", config.Config.PidFile, err)
				time.Sleep(time.Duration(5) * time.Second)
				continue
			}
		}
		if config.Config.LogFile != "" {
			if output, err := os.OpenFile(config.Config.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666); err != nil {
				logrus.Errorf("Failed to log to file [%s] for host-api startup: %v", config.Config.LogFile, err)
				time.Sleep(time.Duration(5) * time.Second)
				continue
			} else {
				logrus.SetOutput(output)
			}
		}
		processor := events.NewDockerEventsProcessor(config.Config.EventsPoolSize)
		err = processor.Process()
		if err != nil {
			logrus.Errorf("Failed to get docker event processor for host-api startup: %v", err)
			time.Sleep(time.Duration(5) * time.Second)
			continue
		}
		break
	}
	for {
		rancherClient, err := util.GetRancherClient()
		if err != nil {
			logrus.Errorf("Failed to get rancher client for host-api startup: %v", err)
			time.Sleep(time.Duration(5) * time.Second)
			continue
		}
		tokenRequest := &rclient.HostApiProxyToken{
			ReportedUuid: config.Config.HostUUID,
		}
		tokenResponse, err := getConnectionToken(0, tokenRequest, rancherClient)
		if err != nil {
			logrus.Errorf("Failed to get connection token for host-api startup: %v", err)
			time.Sleep(time.Duration(5) * time.Second)
			continue
		} else if tokenResponse == nil {
			// nil error and blank token means the proxy is turned off. Just block forever so main function doesn't exit
			var block chan bool
			<-block
		}
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
			logrus.Errorf("Failed to connect to websocket proxy. Error: %v", err)
			time.Sleep(time.Duration(5) * time.Second)
			continue
		}
	}
}

const maxWaitOnHostTries = 20

func getConnectionToken(try int, tokenReq *rclient.HostApiProxyToken, rancherClient *rclient.RancherClient) (*rclient.HostApiProxyToken, error) {
	if try >= maxWaitOnHostTries {
		return nil, errors.New("Reached max retry attempts for getting token")
	}

	tokenResponse, err := rancherClient.HostApiProxyToken.Create(tokenReq)
	if err != nil {
		if apiError, ok := err.(*rclient.ApiError); ok {
			if apiError.StatusCode == 422 {
				m := map[string]string{}
				apiBody := apiError.Body
				parts := strings.Split(apiBody, ", ")
				for _, part := range parts {
					data := strings.Split(part, "=")
					if len(data) == 2 {
						m[data[0]] = data[1]
					}
				}
				if strings.EqualFold(m["code"], "InvalidReference") && strings.EqualFold(m["fieldName"], "reportedUuid") {
					logrus.WithField("reportedUuid", config.Config.HostUUID).WithField("Attempt", try).Infof("Host not registered yet. Sleeping 1 second and trying again.")
					time.Sleep(time.Second)
					try++
					return getConnectionToken(try, tokenReq, rancherClient) // Recursion!
				}
			} else if apiError.StatusCode == 501 {
				logrus.Infof("Host-api proxy disabled. Will not connect.")
				return nil, nil
			}
			return nil, errors.Wrap(err, "Failed to create hostApiProxyToken")
		}
	}
	return tokenResponse, nil
}

type ParsedError struct {
	Code      string
	FieldName string
}
