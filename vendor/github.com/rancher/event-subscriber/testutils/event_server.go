package testUtils

import (
	"io"
	"io/ioutil"
	"net/http"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
)

var subscriberChannels []chan string
var mu sync.RWMutex

var pingHandler func(appData string) error

func SetPingHandler(f func(appData string) error) {
	pingHandler = f
}

func ResetTestServer() {
	mu.Lock()
	defer mu.Unlock()
	for _, channel := range subscriberChannels {
		close(channel)
	}
	subscriberChannels = subscriberChannels[:0]
	pingHandler = nil
}

func publishHandler(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, "A response.")
}

func pushEventHandler(w http.ResponseWriter, req *http.Request) {
	bod, err := ioutil.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	body := string(bod[:])
	pushToSubscribers(body)
}

func subscribeHandler(w http.ResponseWriter, req *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	ws, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade connection.", 500)
		return
	}

	// This is ugly, but it is the only way we can override the ping handler without a big refactor
	if pingHandler != nil {
		ws.SetPingHandler(pingHandler)
	}

	resultChan := make(chan string)
	mu.Lock()
	subscriberChannels = append(subscriberChannels, resultChan)
	mu.Unlock()

	writeEventToSubscriber(ws, resultChan)
}

func pushToSubscribers(message string) {
	mu.RLock()
	defer mu.RUnlock()
	if len(subscriberChannels) > 0 {
		for _, channel := range subscriberChannels {
			log.Printf("sending events: %s", message)
			channel <- message
		}
	}
}

func writeEventToSubscriber(ws *websocket.Conn, c chan string) {
	for {
		event := <-c
		if event != "" {
			err := ws.WriteMessage(websocket.TextMessage, []byte(event))
			if err != nil {
				log.Errorf("Could not write message. Error: %v. Event: %v", err, event)
			}
		}
	}
}

func readyHandler(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, "Ready")
}

func InitializeServer(port string, ready chan string) (err error) {
	http.HandleFunc("/subscribe", subscribeHandler)
	http.HandleFunc("/publish", publishHandler)
	http.HandleFunc("/pushEvent", pushEventHandler)
	http.HandleFunc("/ready", readyHandler)
	go http.ListenAndServe(":"+port, nil)

	readyURL := "http://localhost:" + port + "/pushEvent"
	for {
		resp, err := http.Post(readyURL, "application/json", nil)
		// TODO This was added when I was debuggin. Might not need it now.
		if err == nil {
			log.Println(resp.Status)
			break
		} else {
			log.Fatal(err)
		}
	}

	ready <- "Ready!"
	return nil
}
