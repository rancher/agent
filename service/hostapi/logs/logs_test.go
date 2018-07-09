package logs

import (
	"net/http"
	// "strconv"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/docker/distribution/context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	docker "github.com/docker/docker/client"
	"github.com/gorilla/websocket"
	"github.com/rancher/agent/service/hostapi/config"
	"github.com/rancher/agent/service/hostapi/events"
	"github.com/rancher/agent/service/hostapi/testutils"
	"github.com/rancher/log"
	"github.com/rancher/websocket-proxy/backend"
	"github.com/rancher/websocket-proxy/proxy"
	wsp_utils "github.com/rancher/websocket-proxy/testutils"
	"gopkg.in/check.v1"
)

var privateKey interface{}

func Test(t *testing.T) {
	check.TestingT(t)
}

type LogsTestSuite struct {
	client *docker.Client
}

var _ = check.Suite(&LogsTestSuite{})

func (s *LogsTestSuite) TestCombinedLogs(c *check.C) {
	s.doLogTest(true, "00 ", c)
}

func (s *LogsTestSuite) TestSeparatedLogs(c *check.C) {
	s.doLogTest(false, "01 ", c)
}

func (s *LogsTestSuite) doLogTest(tty bool, prefix string, c *check.C) {
	dialer := &websocket.Dialer{}
	headers := http.Header{}

	newCtr, err := s.client.ContainerCreate(context.Background(), &container.Config{
		Image:     "hello-world:latest",
		OpenStdin: true,
		Tty:       tty,
	}, nil, nil, "logstest")
	if err != nil {
		c.Fatalf("Error creating container, err : [%v]", err)
	}
	err = s.client.ContainerStart(context.Background(), newCtr.ID, types.ContainerStartOptions{})
	if err != nil {
		c.Fatalf("Error starting container, err : [%v]", err)
	}
	defer func() {
		s.client.ContainerRemove(context.Background(), newCtr.ID, types.ContainerRemoveOptions{
			Force:         true,
			RemoveVolumes: true,
		})
	}()

	payload := map[string]interface{}{
		"hostUuid": "1",
		"logs": map[string]interface{}{
			"Container": newCtr.ID,
			"Follow":    true,
		},
	}

	token := wsp_utils.CreateTokenWithPayload(payload, privateKey)
	url := "ws://localhost:3333/v1/logs/?token=" + token
	ws, _, err := dialer.Dial(url, headers)
	if err != nil {
		c.Fatal(err)
	}
	defer ws.Close()

	for count := 0; count < 20; count++ {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			if err == io.EOF {
				return
			}
			c.Fatal(err)
		}
		msgStr := string(msg)
		if !strings.HasPrefix(msgStr, prefix) {
			c.Fatalf("Message didn't have prefix %s: [%s]", prefix, msgStr)
		}
	}
}

func (s *LogsTestSuite) setupWebsocketProxy() {
	config.Parse()
	config.Config.HostUUID = "1"
	config.Config.ParsedPublicKey = wsp_utils.ParseTestPublicKey()
	privateKey = wsp_utils.ParseTestPrivateKey()

	conf := testutils.GetTestConfig(":3333")
	p := &proxy.Starter{
		BackendPaths:  []string{"/v1/connectbackend"},
		FrontendPaths: []string{"/v1/{logs:logs}/"},
		Config:        conf,
	}

	log.Infof("Starting websocket proxy. Listening on [%s], Proxying to cattle API at [%s].",
		conf.ListenAddr, conf.CattleAddr)

	go p.StartProxy()
	time.Sleep(time.Second)
	signedToken := wsp_utils.CreateBackendToken("1", privateKey)

	handlers := make(map[string]backend.Handler)
	handlers["/v1/logs/"] = &Handler{}
	go backend.ConnectToProxy("ws://localhost:3333/v1/connectbackend?token="+signedToken, handlers)
}

func (s *LogsTestSuite) SetUpSuite(c *check.C) {
	cli, err := events.NewDockerClient()
	if err != nil {
		c.Fatalf("Could not connect to docker, err: [%v]", err)
	}
	s.client = cli
	s.pullImage("hello-world", "latest")
	time.Sleep(time.Duration(2) * time.Second)
	s.setupWebsocketProxy()
}

func (s *LogsTestSuite) pullImage(imageRepo, imageTag string) error {
	log.Infof("Pulling %v:%v image.", imageRepo, imageTag)
	_, err := s.client.ImagePull(context.Background(), imageRepo+":"+imageTag, types.ImagePullOptions{})
	return err
}
