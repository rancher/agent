package handlers

import (
	"sync"
)

type ObjWithLock struct {
	mu  sync.Mutex
	obj interface{}
}
