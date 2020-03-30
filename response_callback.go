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
	timeoutTime time.Time ) *responseWithTimeout {
	return &responseWithTimeout{responseCallback: responseCallback,
		timeoutTime: timeoutTime }
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
	return &ResponseCallbackMgr{ responseCallbacks: make(map[int]*responseWithTimeout) }

}

// Add add a response callback for a seqId
func (r *ResponseCallbackMgr) Add(seqId int, callback ResponseCallback, timeoutTime time.Time ) {
	r.Lock()
	defer r.Unlock()
	r.responseCallbacks[seqId] = newResponseWithTimeout(callback, timeoutTime )
}

func (r *ResponseCallbackMgr) Remove(seqId int) (ResponseCallback, bool) {
	r.Lock()
	defer r.Unlock()

	if value, ok := r.responseCallbacks[seqId]; ok {
        delete(r.responseCallbacks, seqId)
		return value.responseCallback, true
	} else {
		return nil, false
	}
}

func (r *ResponseCallbackMgr)getTimeoutResponses() map[int]*responseWithTimeout {
    r.Lock()
    defer r.Unlock()

    timeoutItems := make( map[int]*responseWithTimeout)

    for key, value := range r.responseCallbacks {
        if value.isTimeout() {
            timeoutItems[key] = value
        }
    }
    for key, _:= range timeoutItems {
        delete(r.responseCallbacks, key)
    }

    return timeoutItems

}

func (r *ResponseCallbackMgr) RemoveTimeout(procFunc func(callback ResponseCallback)) {
	timeoutItems := r.getTimeoutResponses()

	for _, value := range timeoutItems {
		procFunc(value.responseCallback)
	}

}
