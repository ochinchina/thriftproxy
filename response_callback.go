package main

import (
	"sync"
	"time"
)

// ResponseCallback the response from the thrift
type ResponseCallback = func(message *Message, err error)

type responseWithTimeout struct {
	responseCallback ResponseCallback
	timeoutTime      time.Time
}

func newResponseWithTimeout(responseCallback ResponseCallback,
	timeout time.Duration) *responseWithTimeout {
	return &responseWithTimeout{responseCallback: responseCallback,
		timeoutTime: time.Now().Add(timeout)}
}

func (r *responseWithTimeout) isTimeout() bool {
	return r.timeoutTime.Before(time.Now())
}

type ResponseCallbackMgr struct {
	sync.Mutex
	responseCallbacks map[int]*responseWithTimeout
}

// NewResponseCallbackMgr create a ResponseCallbackMgr object
func NewResponseCallbackMgr() *ResponseCallbackMgr {
	return &ResponseCallbackMgr{responseCallbacks: make(map[int]*responseWithTimeout)}
}

// Add add a response callback for a seqId
func (r *ResponseCallbackMgr) Add(seqId int, callback ResponseCallback, timeout time.Duration) {
	r.Lock()
	defer r.Unlock()
	r.responseCallbacks[seqId] = newResponseWithTimeout(callback, timeout)
}

func (r *ResponseCallbackMgr) Remove(seqId int) (ResponseCallback, bool) {
	r.Lock()
	defer r.Unlock()

	if value, ok := r.responseCallbacks[seqId]; ok {
		return value.responseCallback, true
	} else {
		return nil, false
	}
}

func (r *ResponseCallbackMgr) RemoveTimeout(procFunc func(callback ResponseCallback)) {
	r.Lock()
	timeoutItems := make(map[int]*responseWithTimeout)
	for key, value := range r.responseCallbacks {
		if value.isTimeout() {
			timeoutItems[key] = value
		}
	}
	for key, _ := range timeoutItems {
		delete(r.responseCallbacks, key)
	}
	r.Unlock()

	for _, value := range timeoutItems {
		procFunc(value.responseCallback)
	}

}
