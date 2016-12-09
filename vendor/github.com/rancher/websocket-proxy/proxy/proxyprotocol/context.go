package proxyprotocol

import (
	"sync"
)

var (
	mutex sync.RWMutex
	infos = make(map[string]*ProxyProtoInfo)
)

func putInfo(clientAddr string, info *ProxyProtoInfo) {
	mutex.Lock()
	defer mutex.Unlock()
	infos[clientAddr] = info
}

func getInfo(clientAddr string) *ProxyProtoInfo {
	mutex.RLock()
	defer mutex.RUnlock()
	return infos[clientAddr]
}

func deleteInfo(clientAddr string) {
	mutex.Lock()
	defer mutex.Unlock()
	delete(infos, clientAddr)
}
