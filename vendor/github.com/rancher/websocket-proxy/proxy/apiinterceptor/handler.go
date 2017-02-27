package apiinterceptor

import (
	"net/http"
	"sync"

	"github.com/pkg/errors"
)

//APIInterceptor is a wrapper over the mux interceptor that does the path<->filters matching
type APIInterceptor struct {
	interceptorRouter http.Handler
	mu                *sync.RWMutex
}

func (h *APIInterceptor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.getRouter().ServeHTTP(w, r)
}

func (h *APIInterceptor) getRouter() http.Handler {
	h.mu.RLock()
	router := h.interceptorRouter
	h.mu.RUnlock()

	return router
}

func (h *APIInterceptor) setRouter(router http.Handler) {
	h.mu.Lock()
	h.interceptorRouter = router
	h.mu.Unlock()
}

func NewInterceptor(configFile string, cattleAddr string) (http.Handler, error) {
	apiInterceptor := &APIInterceptor{
		mu: &sync.RWMutex{},
	}

	router, err := newRouter(configFile, cattleAddr, apiInterceptor)
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't configure api proxy handler")
	}

	apiInterceptor.setRouter(router)
	return apiInterceptor, nil
}

type routerSetter interface {
	setRouter(router http.Handler)
}
