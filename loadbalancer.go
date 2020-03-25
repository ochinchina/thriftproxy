package main

import (
	"errors"
	"sync/atomic"
)

var noBackendAvailable error = errors.New("No backend is available")
var failedAllBackends error = errors.New("Failed on all the backends")

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
	backends    *BackendMgr
	nextBackend int32
}

// NewRoundrobin create a Roundrobin object
func NewRoundrobin() *Roundrobin {
	return &Roundrobin{backends: NewBackendMgr(), nextBackend: 0}
}

// AddBackend add a thrift backend server
func (r *Roundrobin) AddBackend(addr string) {
	if !r.backends.Exists(addr) {
		r.backends.Add(NewBackend(addr))
	}
}

// RemoveBackend remove a previous added thrift backend server
func (r *Roundrobin) RemoveBackend(addr string) error {
	backend, err := r.backends.Remove(addr)
	if err == nil {
		backend.Stop()
	}
	return err
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
