package events

import (
	log "github.com/Sirupsen/logrus"
	"github.com/docker/distribution/context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"time"
)

const workerTimeout = 60 * time.Second

type Handler interface {
	Handle(*events.Message) error
}

type EventRouter struct {
	handlers      map[string][]Handler
	dockerClient  *client.Client
	listener      chan *events.Message
	workers       chan *worker
	workerTimeout time.Duration
	flag          chan bool
}

func NewEventRouter(bufferSize int, workerPoolSize int, dockerClient *client.Client,
	handlers map[string][]Handler) (*EventRouter, error) {
	workers := make(chan *worker, workerPoolSize)
	for i := 0; i < workerPoolSize; i++ {
		workers <- &worker{}
	}

	eventRouter := &EventRouter{
		handlers:      handlers,
		dockerClient:  dockerClient,
		listener:      make(chan *events.Message, bufferSize),
		workers:       workers,
		workerTimeout: workerTimeout,
		flag:          make(chan bool, 1),
	}

	return eventRouter, nil
}

func (e *EventRouter) Start() error {
	log.Info("Starting event router.")
	go e.routeEvents()

	go func() {
	loop:
		for {
			messages, errs := e.dockerClient.Events(context.Background(), types.EventsOptions{})
			select {
			case flag := <-e.flag:
				if flag {
					break loop
				}
			case err := <-errs:
				if err != nil {
					continue loop
				}
			case m := <-messages:
				e.listener <- &m
			}
		}
	}()
	return nil
}

func (e *EventRouter) Stop() error {
	if e.listener == nil {
		return nil
	}
	e.flag <- true
	return nil
}

func (e *EventRouter) routeEvents() {
	for {
		event := <-e.listener
		timer := time.NewTimer(e.workerTimeout)
		gotWorker := false
		for !gotWorker {
			select {
			case w := <-e.workers:
				go w.doWork(event, e)
				gotWorker = true
			case <-timer.C:
				log.Infof("Timed out waiting for worker. Re-initializing wait.")
			}
		}
	}
}

type worker struct{}

func (w *worker) doWork(event *events.Message, e *EventRouter) {
	defer func() { e.workers <- w }()
	if event == nil {
		return
	}
	if handlers, ok := e.handlers[event.Status]; ok {
		log.Debugf("Processing event: %#v", event)
		for _, handler := range handlers {
			if err := handler.Handle(event); err != nil {
				log.Errorf("Error processing event %#v. Error: %v", event, err)
			}
		}
	}
}
