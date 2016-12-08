package stats

import (
	"flag"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"

	"github.com/docker/distribution/context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/rancher/agent/service/hostapi/config"
	"github.com/rancher/agent/service/hostapi/events"
	"github.com/rancher/agent/service/hostapi/testutils"
	"github.com/rancher/websocket-proxy/backend"
	"github.com/rancher/websocket-proxy/proxy"
	wsp_utils "github.com/rancher/websocket-proxy/testutils"
)

var privateKey interface{}

func TestContainerStats(t *testing.T) {
	dialer := &websocket.Dialer{}
	headers := http.Header{}
	c, err := events.NewDockerClient()
	if err != nil {
		t.Fatalf("Could not connect to docker, err: [%v]", err)
	}
	allCtrs, err := c.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		t.Fatalf("Error listing all images, err : [%v]", err)
	}
	ctrs := []string{}
	for _, ctr := range allCtrs {
		if strings.HasPrefix(ctr.Image, "busybox:1") {
			ctrs = append(ctrs, ctr.ID)
		}
	}
	if len(ctrs) != 1 {
		t.Fatalf("Expected 1 containers, but got %v: [%v]", len(ctrs), ctrs)
	}

	cIds := map[string]string{}
	payload := map[string]interface{}{
		"hostUuid":     "1",
		"containerIds": cIds,
	}

	for i, ctr := range ctrs {
		cIds[ctr] = "1i" + strconv.Itoa(i+1)
	}

	log.Infof("%+v", cIds)
	time.Sleep(2 * time.Second)

Outer:
	for i := 0; i < 5; i++ {
		token := wsp_utils.CreateTokenWithPayload(payload, privateKey)
		url := "ws://localhost:1111/v1/containerstats?token=" + token
		ws, _, err := dialer.Dial(url, headers)
		if err != nil {
			t.Fatal(err)
		}
		defer ws.Close()

		for count := 0; count < 4; count++ {
			_, msg, err := ws.ReadMessage()
			if err == io.EOF {
				// May take a second or two before cadvisor knows about the container
				time.Sleep(500 * time.Millisecond)
				continue Outer
			}
			if err != nil {
				t.Fatal(err)
			}
			stats := string(msg)
			if !strings.Contains(stats, "1i1") {
				t.Fatalf("Stats are not working. Output: [%s]", stats)
			}
		}
		return
	}

	log.Fatal(io.EOF)
}

// This test wont work in dind. Disabling it for now, until I figure out a solution
func unTestContainerStatSingleContainer(t *testing.T) {
	dialer := &websocket.Dialer{}
	headers := http.Header{}

	c, err := events.NewDockerClient()
	if err != nil {
		t.Fatalf("Could not connect to docker, err: [%v]", err)
	}

	filter := filters.NewArgs()
	filter.Add("image", "")
	ctrs, err := c.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil || len(ctrs) == 0 {
		t.Fatalf("Error listing all images, err : [%v]", err)
	}
	payload := map[string]interface{}{
		"hostUuid": "1",
		"containerIds": map[string]string{
			ctrs[0].ID: "1i1",
		},
	}

	log.Info(ctrs[0].ID)

	token := wsp_utils.CreateTokenWithPayload(payload, privateKey)
	url := "ws://localhost:1111/v1/containerstats/" + ctrs[0].ID + "?token=" + token
	ws, _, err := dialer.Dial(url, headers)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	for count := 0; count < 4; count++ {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		stats := string(msg)
		if !strings.Contains(stats, "1i1") {
			t.Fatalf("Stats are not working. Output: [%s]", stats)
		}
	}
}

func TestHostStats(t *testing.T) {
	dialer := &websocket.Dialer{}
	headers := http.Header{}

	payload := map[string]interface{}{
		"hostUuid":   "1",
		"resourceId": "1h1",
	}

	token := wsp_utils.CreateTokenWithPayload(payload, privateKey)
	url := "ws://localhost:1111/v1/hoststats?token=" + token
	ws, _, err := dialer.Dial(url, headers)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	for count := 0; count < 4; count++ {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		stats := string(msg)
		if !strings.Contains(stats, "1h1") {
			t.Fatalf("Stats are not working. Output: [%s]", stats)
		}
	}
}

func TestHostStatsLegacy(t *testing.T) {
	dialer := &websocket.Dialer{}
	headers := http.Header{}
	token := wsp_utils.CreateToken("1", privateKey)
	url := "ws://localhost:1111/v1/stats?token=" + token
	ws, _, err := dialer.Dial(url, headers)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	count := 0
	for {
		if count > 3 {
			break
		}

		_, msg, err := ws.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		stats := string(msg)
		if !strings.Contains(stats, "cpu") {
			t.Fatalf("Stats are not working. Output: [%s]", stats)
		}
		count++
	}
}

func setupWebsocketProxy() {
	config.Parse()
	config.Config.NumStats = 1
	config.Config.ParsedPublicKey = wsp_utils.ParseTestPublicKey()
	privateKey = wsp_utils.ParseTestPrivateKey()

	conf := testutils.GetTestConfig(":1111")
	p := &proxy.Starter{
		BackendPaths:  []string{"/v1/connectbackend"},
		FrontendPaths: []string{"/v1/{logs:logs}/", "/v1/{stats:stats}", "/v1/{stats:stats}/{statsid}", "/v1/exec/"},
		StatsPaths: []string{"/v1/{hoststats:hoststats(\\/project)?(\\/)?}",
			"/v1/{containerstats:containerstats(\\/service)?(\\/)?}",
			"/v1/{containerstats:containerstats}/{containerid}"},
		Config: conf,
	}

	log.Infof("Starting websocket proxy. Listening on [%s], Proxying to cattle API at [%s].",
		conf.ListenAddr, conf.CattleAddr)

	go p.StartProxy()
	time.Sleep(time.Second)
	signedToken := wsp_utils.CreateBackendToken("1", privateKey)

	handlers := make(map[string]backend.Handler)
	handlers["/v1/stats/"] = &Handler{}
	handlers["/v1/hoststats/"] = &HostStatsHandler{}
	handlers["/v1/containerstats/"] = &ContainerStatsHandler{}
	go backend.ConnectToProxy("ws://localhost:1111/v1/connectbackend?token="+signedToken, handlers)
	time.Sleep(300 * time.Millisecond)
}

func TestMain(m *testing.M) {
	flag.Parse()
	setupWebsocketProxy()
	os.Exit(m.Run())
}
