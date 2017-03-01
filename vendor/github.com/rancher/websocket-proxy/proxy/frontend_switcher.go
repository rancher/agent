package proxy

import (
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
)

type Switcher struct {
	sync.Mutex
	remote http.Handler
	config *Config
	addr   string
}

func NewSwitcher(config *Config) *Switcher {
	s := &Switcher{
		config: config,
	}
	go s.start()
	return s
}

func (s *Switcher) Wrap(local http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if handler := s.lookupHandler(); handler != nil {
			handler.ServeHTTP(rw, req)
		} else {
			local.ServeHTTP(rw, req)
		}
	})
}

func (s *Switcher) lookupHandler() http.Handler {
	var result http.Handler
	s.Lock()
	result = s.remote
	s.Unlock()
	return result
}

func (s *Switcher) start() {
	logrus.Infof("Master config file: %s", s.config.MasterFile)
	if s.config.MasterFile == "" {
		s.clear()
		return
	}

	for {
		if err := s.readConfig(); err != nil {
			logrus.Errorf("Failed to read config: %v", err)
		}
		time.Sleep(5 * time.Second)
	}
}

func (s *Switcher) readConfig() error {
	bytes, err := ioutil.ReadFile(s.config.MasterFile)
	if os.IsNotExist(err) {
		s.clear()
		return nil
	} else if err != nil {
		return err
	}

	newAddr := strings.TrimSpace(string(bytes))

	if newAddr == "" {
		s.clear()
		return nil
	}

	if s.addr == newAddr {
		return nil
	}

	s.Lock()
	logrus.Infof("Master address: %s", newAddr)
	s.addr = newAddr
	s.remote = newWSProxy(&Config{
		CattleAddr: newAddr,
	})
	s.Unlock()

	return nil
}

func (s *Switcher) clear() {
	if s.remote == nil {
		return
	}
	s.Lock()
	logrus.Infof("Master address: none, using local")
	s.remote = nil
	s.Unlock()
}
