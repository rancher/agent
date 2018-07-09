package framework

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rancher/log"
)

func NewServer() *mux.Router {
	s := &server{}

	router := mux.NewRouter().StrictSlash(true)
	router.Methods("POST").Path("/events").HandlerFunc(s.PostEvent)
	router.Methods("GET").Path("/die").HandlerFunc(s.Die)
	router.Methods("GET").Path("/ping").HandlerFunc(s.Ping)

	return router
}

type server struct {
}

func (s *server) Ping(rw http.ResponseWriter, req *http.Request) {
	// just want to return a 200
}

func (s *server) Die(rw http.ResponseWriter, req *http.Request) {
	log.Fatal("Instructed to die.")
}

func (s *server) PostEvent(rw http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	response := testEvent(body)
	js, err := json.Marshal(response)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.Write(js)
}

type responseError struct {
	Message string
	Code    string
	Status  int
}
