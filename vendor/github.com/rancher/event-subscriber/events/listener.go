package events

import (
	"bytes"
	"encoding/base64"
	"encoding/json"

	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
	"github.com/rancher/go-rancher/v3"
)

const MaxWait = time.Duration(time.Second * 10)

// EventHandler Defines the function "interface" that handlers must conform to.
type EventHandler func(*Event, *client.RancherClient) error

type EventRouter struct {
	name          string
	priority      int
	apiURL        string
	accessKey     string
	secretKey     string
	apiClient     *client.RancherClient
	subscribeURL  string
	eventHandlers map[string]EventHandler
	workerCount   int
	eventStream   *websocket.Conn
	resourceName  string
	pingConfig    PingConfig
}

func NewEventRouter(name string, priority int, apiURL string, accessKey string, secretKey string,
	apiClient *client.RancherClient, eventHandlers map[string]EventHandler, resourceName string, workerCount int,
	pingConfig PingConfig) (*EventRouter, error) {

	if apiClient == nil {
		var err error
		apiClient, err = client.NewRancherClient(&client.ClientOpts{
			Timeout:   time.Second * 30,
			Url:       apiURL,
			AccessKey: accessKey,
			SecretKey: secretKey,
		})
		if err != nil {
			return nil, err
		}
	}

	// TODO Get subscribe collection URL from API instead of hard coding
	subscribeURL := strings.Replace(apiURL+"/subscribe", "http", "ws", 1)

	return &EventRouter{
		name:          name,
		priority:      priority,
		apiURL:        apiURL,
		accessKey:     accessKey,
		secretKey:     secretKey,
		apiClient:     apiClient,
		subscribeURL:  subscribeURL,
		eventHandlers: eventHandlers,
		workerCount:   workerCount,
		resourceName:  resourceName,
		pingConfig:    pingConfig,
	}, nil
}

// The difference between Start and StartWithoutCreate is a matter of making this event router
// more generally usable. The Start implementation creates
// the necessary ExternalHandler upon start up. This router has been refactor to
// be used in situations where creating an externalHandler is not desired.
// This allows the router to be used for Agent connections and for ExternalHandlers
// that are created outside of this router.

func (router *EventRouter) Start(ready chan<- bool) error {
	eventSuffix := ";handler=" + router.name
	wp := SkippingWorkerPool(router.workerCount, resourceIDLocker)
	return router.run(wp, ready, eventSuffix)
}

func (router *EventRouter) StartWithoutCreate(ready chan<- bool) error {
	wp := SkippingWorkerPool(router.workerCount, resourceIDLocker)
	return router.run(wp, ready, "")
}

func (router *EventRouter) RunWithWorkerPool(wp WorkerPool) error {
	return router.run(wp, nil, "")
}

func (router *EventRouter) run(wp WorkerPool, ready chan<- bool, eventSuffix string) (err error) {

	log.WithFields(log.Fields{
		"workerCount": router.workerCount,
	}).Info("Initializing event router")

	handlers := map[string]EventHandler{}

	if pingHandler, ok := router.eventHandlers["ping"]; ok {
		// Ping doesnt need registered in the POST and ping events don't have the handler suffix.
		//If we start handling other non-suffix events, we might consider improving this.
		handlers["ping"] = pingHandler
	}

	subscribeParams := url.Values{}
	for event, handler := range router.eventHandlers {
		fullEventKey := event + eventSuffix
		subscribeParams.Add("eventNames", fullEventKey)
		handlers[fullEventKey] = handler
	}

	eventStream, err := router.subscribeToEvents(router.subscribeURL, router.accessKey, router.secretKey, subscribeParams)
	if err != nil {
		return err
	}
	log.Info("Connection established")
	router.eventStream = eventStream
	defer router.Stop()

	if ready != nil {
		ready <- true
	}

	ph := newPongHandler(router)
	defer ph.stop()
	router.eventStream.SetPongHandler(ph.handle)
	go router.sendWebsocketPings()

	for {
		_, message, err := router.eventStream.ReadMessage()
		if err != nil {
			// Error here means the connection is closed. It's normal, so just return.
			return nil
		}

		message = bytes.TrimSpace(message)
		if len(message) == 0 {
			continue
		}

		event := &Event{}
		err = json.Unmarshal(message, &event)
		if err != nil {
			log.WithFields(log.Fields{
				"message": string(message),
			}).Warnf("Error parsing message: %s", err)
			continue
		}
		wp.HandleWork(event, handlers, router.apiClient)
	}
}

func (router *EventRouter) Stop() {
	router.eventStream.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	router.eventStream.Close()
}

func (router *EventRouter) subscribeToEvents(subscribeURL string, accessKey string, secretKey string, data url.Values) (*websocket.Conn, error) {
	dialer := &websocket.Dialer{}
	headers := http.Header{}
	headers.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(accessKey+":"+secretKey)))
	subscribeURL = subscribeURL + "?" + data.Encode()
	ws, resp, err := dialer.Dial(subscribeURL, headers)

	if err != nil {
		log.WithFields(log.Fields{
			"subscribeUrl": subscribeURL,
		}).Errorf("Error subscribing to events: %s", err)
		if resp != nil {
			log.WithFields(log.Fields{
				"status":          resp.Status,
				"statusCode":      resp.StatusCode,
				"responseHeaders": resp.Header,
			}).Error("Got error response")
			if resp.Body != nil {
				defer resp.Body.Close()
				body, _ := ioutil.ReadAll(resp.Body)
				log.Errorf("Error response: %s", body)
			}
		}
		return nil, err
	}
	return ws, nil
}

func (router *EventRouter) GetWebSocketConn() *websocket.Conn {
	return router.eventStream
}
