package main

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"sync/atomic"
)

var noBackendAvailable error = errors.New("No backend is available")
var failedAllBackends error = errors.New("Failed on all the backends")

// LoadBalancer
type LoadBalancer interface {
	// add a backend
	AddBackend(backendInfo *BackendInfo)

	// remove previous added backend
	RemoveBackend(addr string) error

	// send a message to thrift server
	Send(msg *Message, callback ResponseCallback)
}

// Roundrobin this class implements LoadBalancer interface
type Roundrobin struct {
	resolver    *Resolver
	backends    *BackendMgr
	nextBackend int32
}

// NewRoundrobin create a Roundrobin object
func NewRoundrobin() *Roundrobin {
	return &Roundrobin{resolver: NewResolver(10),
		backends:    NewBackendMgr(),
		nextBackend: 0}
}

// AddBackend add a thrift backend server
func (r *Roundrobin) AddBackend(backendInfo *BackendInfo) {
	hostname, _, err := splitAddr(backendInfo.Addr)

	if err != nil {
		log.WithFields(log.Fields{"address": backendInfo.Addr}).Error("Backend address is invalid")
		return
	}

	log.WithFields(log.Fields{"address": backendInfo.Addr}).Info("Add backend")

	if !isIPAddress(hostname) {
		r.resolver.ResolveHost(backendInfo.Addr, func(hostname string, newAddrs []string, removedAddrs []string) {
			r.resolvedAddrs(hostname, newAddrs, removedAddrs, backendInfo.Readiness)
		})
	} else if !r.backends.Exists(backendInfo.Addr) {
		r.backends.Add(NewBackend(backendInfo.Addr, createReadiness(backendInfo.Addr, backendInfo.Readiness)))
	}
}

func (r *Roundrobin) resolvedAddrs(hostname string, newAddrs []string, removedAddrs []string, readinessConf *ReadinessConf) {
	for _, addr := range newAddrs {
		r.AddBackend(&BackendInfo{Addr: addr, Readiness: readinessConf})
	}
	for _, addr := range removedAddrs {
		r.RemoveBackend(addr)
	}
}

// RemoveBackend remove a previous added thrift backend server
func (r *Roundrobin) RemoveBackend(addr string) error {
	hostname, _, err := splitAddr(addr)

	if err != nil {
		return err
	}
	if !isIPAddress(hostname) {
		ips := r.resolver.GetAddrsOfHost(addr)
		r.resolver.StopResolve(addr)
		for _, a := range ips {
			r.RemoveBackend(a)
		}
		return nil
	} else {
		backend, err := r.backends.Remove(addr)
		if err == nil {
			backend.Stop()
		}
		return err
	}
}

// Send send a request to one of thrift backend server
func (r *Roundrobin) Send(request *Message, callback ResponseCallback) {
	n := int32(r.backends.Size())
	if n <= 0 {
		callback(nil, noBackendAvailable)
	} else {
		index := atomic.AddInt32(&r.nextBackend, int32(1)) % n
		r.sendTo(request, index, n, n, callback)
	}
}

func (r *Roundrobin) sendTo(request *Message, index int32, leftTimes int32, total int32, callback ResponseCallback) {
	if leftTimes <= 0 {
		callback(nil, failedAllBackends)
	} else {
		backend, err := r.backends.GetIndex(int(index))
		if err == nil {
			backend.Send(request, func(response *Message, err error) {
				if err == nil {
					callback(response, err)
				} else {
					r.sendTo(request, (index+1)%total, leftTimes-1, total, callback)
				}
			})
		} else {
			r.sendTo(request, (index+1)%total, leftTimes-1, total, callback)
		}
	}
}
