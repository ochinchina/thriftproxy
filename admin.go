package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
)

type Admin struct {
	server   http.Server
	proxyMgr *ProxyMgr
}

type BackendInfo struct {
	Proxies []struct {
		Name     string
		Backends []string
	}
}

func NewAdmin(addr string, proxyMgr *ProxyMgr) *Admin {
	admin := &Admin{proxyMgr: proxyMgr}
	admin.server.Addr = addr
	router := mux.NewRouter()
	router.HandleFunc("/addbackend", admin.processAddBackend)
	router.HandleFunc("/removebackend", admin.processRemoveBackend)
	admin.server.Handler = router
	return admin
}

func (admin *Admin) Start() {
	go admin.server.ListenAndServe()
}

func (admin *Admin) processAddBackend(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	defer r.Body.Close()
	backendInfo, err := admin.readBackendInfo(r)
	if err == nil {
		admin.addBackend(backendInfo)
	}
}

func (admin *Admin) processRemoveBackend(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	defer r.Body.Close()
	backendInfo, err := admin.readBackendInfo(r)
	if err == nil {
		admin.removeBackend(backendInfo)
	}
}

func (admin *Admin) readBackendInfo(r *http.Request) (*BackendInfo, error) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	info := BackendInfo{}
	err = json.Unmarshal(b, &info)
	return &info, err

}

func (admin *Admin) processBackend(backendInfo *BackendInfo, proxyProcFunc func(proxy *Proxy, addr string)) {
	for _, proxyInfo := range backendInfo.Proxies {
		proxy, err := admin.proxyMgr.GetProxy(proxyInfo.Name)
		if err == nil {
			for _, backend := range proxyInfo.Backends {
				proxyProcFunc(proxy, backend)
			}
		}
	}

}

func (admin *Admin) addBackend(backendInfo *BackendInfo) {
	admin.processBackend(backendInfo, func(proxy *Proxy, addr string) {
		proxy.AddBackend(addr)
	})
}

func (admin *Admin) removeBackend(backendInfo *BackendInfo) {
	admin.processBackend(backendInfo, func(proxy *Proxy, addr string) {
		proxy.RemoveBackend(addr)
	})
}
