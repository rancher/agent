package main

import (
	log "github.com/Sirupsen/logrus"

	"github.com/rancher/websocket-proxy/proxy"
)

func main() {

	conf, err := proxy.GetConfig()
	if err != nil {
		log.WithField("error", err).Fatal("Error getting config.")
	}

	p := &proxy.Starter{
		BackendPaths: []string{
			"/v1/connectbackend",
			"/v2-beta/connectbackend",
			"/v2/connectbackend",
		},
		FrontendPaths: []string{
			"/v1/{logs:logs}",
			"/v1/{logs:logs}/",
			"/v1/{stats:stats}",
			"/v1/{stats:stats}/{statsid}",
			"/v1/exec/",
			"/v1/exec",
			"/v1/console/",
			"/v1/console",
			"/v1/dockersocket/",
			"/v1/dockersocket",
			"/v2-beta/{logs:logs}/",
			"/v2-beta/{logs:logs}",
			"/v2-beta/{stats:stats}",
			"/v2-beta/{stats:stats}/{statsid}",
			"/v2-beta/exec/",
			"/v2-beta/exec",
			"/v2-beta/console/",
			"/v2-beta/console",
			"/v2-beta/dockersocket/",
			"/v2-beta/dockersocket",
			"/v2/{logs:logs}/",
			"/v2/{logs:logs}",
			"/v2/{stats:stats}",
			"/v2/{stats:stats}/{statsid}",
			"/v2/exec/",
			"/v2/exec",
			"/v2/console/",
			"/v2/console",
			"/v2/dockersocket/",
			"/v2/dockersocket",
		},
		FrontendHTTPPaths: []string{
			"/v1/container-proxy{path:.*}",
			"/v2-beta/container-proxy{path:.*}",
			"/v2/container-proxy{path:.*}",
			"/r/projects/{project}/{service}{path:.*}",
			"/r/{service}{path:.*}",
		},
		StatsPaths: []string{
			"/v1/{hoststats:hoststats(\\/project)?(\\/)?}",
			"/v1/{containerstats:containerstats(\\/service)?(\\/)?}",
			"/v1/{containerstats:containerstats}/{containerid}",
			"/v2-beta/{hoststats:hoststats(\\/project)?(\\/)?}",
			"/v2-beta/{containerstats:containerstats(\\/service)?(\\/)?}",
			"/v2-beta/{containerstats:containerstats}/{containerid}",
			"/v2/{hoststats:hoststats(\\/project)?(\\/)?}",
			"/v2/{containerstats:containerstats(\\/service)?(\\/)?}",
			"/v2/{containerstats:containerstats}/{containerid}",
		},
		CattleWSProxyPaths: []string{
			"/v1/{sub:subscribe}",
			"/v1/projects/{project}/{sub:subscribe}",
			"/v2-beta/{sub:subscribe}",
			"/v2-beta/projects/{project}/{sub:subscribe}",
			"/v2/{sub:subscribe}",
			"/v2/projects/{project}/{sub:subscribe}",
		},
		CattleProxyPaths: []string{
			"/{cattle-proxy:.*}",
		},
		Config: conf,
	}

	log.Infof("Starting websocket proxy. Listening on [%s], Proxying to cattle API at [%s], Monitoring parent pid [%v].",
		conf.ListenAddr, conf.CattleAddr, conf.ParentPid)

	err = p.StartProxy()

	log.WithFields(log.Fields{"error": err}).Info("Exiting proxy.")
}
