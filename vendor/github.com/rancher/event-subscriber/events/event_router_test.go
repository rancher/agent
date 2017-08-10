package events

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	tu "github.com/rancher/event-subscriber/testutils"
	"github.com/rancher/go-rancher/v3"
)

const eventServerPort string = "8005"
const baseURL string = "http://localhost:" + eventServerPort
const pushURL string = baseURL + "/pushEvent"

func newRouter(eventHandlers map[string]EventHandler, workerCount int, t *testing.T, pingConfig PingConfig) *EventRouter {
	fakeAPIClient := &client.RancherClient{}
	router, err := NewEventRouter("testRouter", 2000, baseURL, "accKey", "secret", fakeAPIClient,
		eventHandlers, "physicalhost", workerCount, pingConfig)
	if err != nil {
		t.Fatal(err)
	}
	return router
}

func TestWebsocketPingTimeout(t *testing.T) {
	testHandler := func(event *Event, apiClient *client.RancherClient) error {
		return nil
	}

	tu.SetPingHandler(func(appData string) error {
		return nil
	})

	eventHandlers := map[string]EventHandler{"physicalhost.create": testHandler}
	router := newRouter(eventHandlers, 3, t, PingConfig{
		SendPingInterval:  500,
		CheckPongInterval: 500,
		MaxPongWait:       1000,
	})
	ready := make(chan bool, 1)

	routerStopped := make(chan bool, 1)
	go func() {
		router.Start(ready)
		routerStopped <- true
	}()

	defer tu.ResetTestServer()

	// Wait for start to be ready
	<-ready

	select {
	case <-routerStopped:
		// Successfully shutdown after missed pings
	case <-time.After(time.Millisecond * 1500):
		t.Fatalf("Router did not stop because of failed pings.")
	}

}

// Tests the simplest case of successfully receiving, routing, and handling
// three events.
func TestSimpleRouting(t *testing.T) {
	eventsReceived := make(chan *Event)
	testHandler := func(event *Event, apiClient *client.RancherClient) error {
		eventsReceived <- event
		return nil
	}

	eventHandlers := map[string]EventHandler{"physicalhost.create": testHandler}
	router := newRouter(eventHandlers, 3, t, DefaultPingConfig)
	ready := make(chan bool, 1)
	go router.Start(ready)
	defer router.Stop()
	defer tu.ResetTestServer()
	// Wait for start to be ready
	<-ready

	preCount := 0
	pre := func(event *Event) {
		event.ID = strconv.Itoa(preCount)
		event.ResourceID = strconv.FormatInt(time.Now().UnixNano(), 10)
		preCount++
		event.Name = "physicalhost.create;handler=testRouter"
	}

	// Push 3 events
	for i := 0; i < 3; i++ {
		err := prepAndPostEvent("../testutils/resources/machine_create_event.json", pre)
		if err != nil {
			t.Fatal(err)
		}
	}
	receivedEvents := map[string]*Event{}
	for i := 0; i < 3; i++ {
		receivedEvent := awaitEvent(eventsReceived, 100, t)
		if receivedEvent != nil {
			receivedEvents[receivedEvent.ID] = receivedEvent
		}
	}

	for i := 0; i < 3; i++ {
		if _, ok := receivedEvents[strconv.Itoa(i)]; !ok {
			t.Errorf("Didn't get event %v", i)
		}
	}
}

// If no workers are available (because they're all busy), an event should simply be dropped.
// This tests that functionality
func TestEventDropping(t *testing.T) {
	eventsReceived := make(chan *Event)
	stopWaiting := make(chan bool)
	testHandler := func(event *Event, apiClient *client.RancherClient) error {
		eventsReceived <- event
		<-stopWaiting
		return nil
	}

	eventHandlers := map[string]EventHandler{"physicalhost.create": testHandler}

	// 2 workers, not 3, means the last event should be droppped
	router := newRouter(eventHandlers, 2, t, DefaultPingConfig)
	ready := make(chan bool, 1)
	go router.Start(ready)
	defer router.Stop()
	defer tu.ResetTestServer()
	// Wait for start to be ready
	<-ready

	preCount := 0
	pre := func(event *Event) {
		event.ID = strconv.Itoa(preCount)
		event.ResourceID = strconv.FormatInt(time.Now().UnixNano(), 10)
		preCount++
		event.Name = "physicalhost.create;handler=testRouter"
	}

	// Push 3 events
	for i := 0; i < 3; i++ {
		err := prepAndPostEvent("../testutils/resources/machine_create_event.json", pre)
		if err != nil {
			t.Fatal(err)
		}
	}
	receivedEvents := map[string]*Event{}
	for i := 0; i < 3; i++ {
		receivedEvent := awaitEvent(eventsReceived, 20, t)
		if receivedEvent != nil {
			receivedEvents[receivedEvent.ID] = receivedEvent
		}
	}

	if len(receivedEvents) != 2 {
		t.Errorf("Unexpected length %v", len(receivedEvents))
	}
}

// Tests that when we have more events than workers, workers are added back to the pool
// when they are done doing their work and capable of handling more work.
func TestWorkerReuse(t *testing.T) {
	eventsReceived := make(chan *Event)
	testHandler := func(event *Event, apiClient *client.RancherClient) error {
		time.Sleep(10 * time.Millisecond)
		eventsReceived <- event
		return nil
	}

	eventHandlers := map[string]EventHandler{"physicalhost.create": testHandler}

	router := newRouter(eventHandlers, 1, t, DefaultPingConfig)
	ready := make(chan bool, 1)
	go router.Start(ready)
	defer router.Stop()
	defer tu.ResetTestServer()
	// Wait for start to be ready
	<-ready
	preCount := 1
	pre := func(event *Event) {
		event.ID = strconv.Itoa(preCount)
		event.ResourceID = strconv.FormatInt(time.Now().UnixNano(), 10)
		preCount++
		event.Name = "physicalhost.create;handler=testRouter"
	}

	// Push 3 events
	receivedEvents := map[string]*Event{}
	for i := 0; i < 2; i++ {
		err := prepAndPostEvent("../testutils/resources/machine_create_event.json", pre)
		if err != nil {
			t.Fatal(err)
		}
		receivedEvent := awaitEvent(eventsReceived, 500, t)
		if receivedEvent != nil {
			receivedEvents[receivedEvent.ID] = receivedEvent
		}
	}

	if len(receivedEvents) != 2 {
		t.Errorf("Unexpected length %v", len(receivedEvents))
	}
}

func awaitEvent(eventsReceived chan *Event, millisToWait int, t *testing.T) *Event {
	timeout := make(chan bool, 1)
	timeoutFunc := func() {
		time.Sleep(time.Duration(millisToWait) * time.Millisecond)
		timeout <- true
	}
	go timeoutFunc()

	select {
	case e := <-eventsReceived:
		return e
	case <-timeout:
		return nil
	}
}

type PreFunc func(*Event)

func prepAndPostEvent(eventFile string, preFunc PreFunc) (err error) {
	rawEvent, err := ioutil.ReadFile(eventFile)
	if err != nil {
		return err
	}

	event := &Event{}
	err = json.Unmarshal(rawEvent, &event)
	if err != nil {
		return err
	}
	preFunc(event)
	rawEvent, err = json.Marshal(event)
	if err != nil {
		return err
	}

	buffer := new(bytes.Buffer)
	err = json.Compact(buffer, rawEvent)
	if err != nil {
		return err
	}
	http.Post(pushURL, "application/json", buffer)

	return nil
}

func TestMain(m *testing.M) {
	ready := make(chan string, 1)
	go tu.InitializeServer(eventServerPort, ready)
	<-ready
	result := m.Run()
	os.Exit(result)
}
