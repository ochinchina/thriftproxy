package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"net/http"
)

type Admin struct {
	server   http.Server
	proxyMgr *ProxyMgr
}

type ProxyBackends struct {
	Proxies []struct {
		Name     string
		Backends []BackendInfo
	}
}

func NewAdmin(addr string, proxyMgr *ProxyMgr) *Admin {
	admin := &Admin{proxyMgr: proxyMgr}
	admin.server.Addr = addr
	router := mux.NewRouter()
	router.HandleFunc("/backends/add", admin.processAddBackend)
	router.HandleFunc("/backends/remove", admin.processRemoveBackend)
	router.HandleFunc("/backends/list", admin.processGetBackends)
	admin.server.Handler = router
	return admin
}

func (admin *Admin) Start() {
	go admin.server.ListenAndServe()
}

func (admin *Admin) processAddBackend(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	proxyBackends, err := admin.readProxyBackends(r)
	if err == nil {
		admin.addBackend(proxyBackends)
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (admin *Admin) processRemoveBackend(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	proxyBackends, err := admin.readProxyBackends(r)
	if err == nil {
		admin.removeBackend(proxyBackends)
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (admin *Admin) processGetBackends(w http.ResponseWriter, r *http.Request) {
	result := admin.getAllBackends()
	b, err := json.Marshal(result)
	if err == nil {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "application/json")
		w.Write(b)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Fail to encode the backends as json"))
	}
}

func (admin *Admin) getAllBackends() map[string][]interface{} {
	allProxy := admin.proxyMgr.GetAllProxy()
	result := make(map[string][]interface{})
	for _, proxy := range allProxy {
		backends := make([]interface{}, 0)
		for _, backend := range proxy.GetAllBackends() {
			addr := backend.GetAddr()
			connected := backend.IsConnected()
			backendInfo := struct {
				Addr      string
				Connected bool
			}{addr, connected}
			backends = append(backends, &backendInfo)

		}
		result[proxy.GetName()] = backends
	}
	return result

}

func (admin *Admin) readProxyBackends(r *http.Request) (*ProxyBackends, error) {
	proxyBackends := &ProxyBackends{}
	decoder := yaml.NewDecoder(r.Body)
	err := decoder.Decode(proxyBackends)
	if err != nil {
		return nil, err
	}
	return proxyBackends, nil
}

func (admin *Admin) processBackend(proxyBackends *ProxyBackends, proxyProcFunc func(proxy *Proxy, backend *BackendInfo)) {
	for _, proxyInfo := range proxyBackends.Proxies {
		proxy, err := admin.proxyMgr.GetProxy(proxyInfo.Name)
		if err == nil {
			for _, backend := range proxyInfo.Backends {
				proxyProcFunc(proxy, &backend)
			}
		} else {
			log.WithFields(log.Fields{"proxy": proxyInfo.Name}).Error("fail to find the proxy by name")
		}
	}

}

func (admin *Admin) addBackend(proxyBackends *ProxyBackends) {
	admin.processBackend(proxyBackends, func(proxy *Proxy, backend *BackendInfo) {
		proxy.AddBackend(backend)
	})
}

func (admin *Admin) removeBackend(proxyBackends *ProxyBackends) {
	admin.processBackend(proxyBackends, func(proxy *Proxy, backend *BackendInfo) {
		proxy.RemoveBackend(backend.Addr)
	})
}
