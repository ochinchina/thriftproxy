package main

import (
	"errors"
	"strings"
	"sync/atomic"
)

var noBackendAvailable error = errors.New("No backend is available")
var failedAllBackends error = errors.New("Failed on all the backends")

func splitAddr(addr string) (hostname string, port string, err error) {
	pos := strings.LastIndex(addr, ":")
	if pos == -1 {
		err = errors.New("not a valid address")
	} else {
		hostname = addr[0:pos]
		port = addr[pos+1:]
		err = nil
	}
	return

}

// LoadBalancer
type LoadBalancer interface {
	// add a backend, the address is in ip:port format
	AddBackend(addr string)

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
func (r *Roundrobin) AddBackend(addr string) {
	hostname, _, err := splitAddr(addr)

	if err != nil {
		return
	}

	if !isIPAddress(hostname) {
		r.resolver.ResolveHost(addr, r.resolvedAddrs)
	} else if !r.backends.Exists(addr) {
		r.backends.Add(NewBackend(addr))
	}
}

func (r *Roundrobin) resolvedAddrs(hostname string, newAddrs []string, removedAddrs []string) {
	for _, addr := range newAddrs {
		r.AddBackend(addr)
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
