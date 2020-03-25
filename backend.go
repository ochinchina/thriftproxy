package main

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"net"
	"sync/atomic"
	"time"
)

var notConnectedError error = errors.New("not connected")

type requestWithResponseCallback struct {
	request          *Message
	responseCallback ResponseCallback
}

func newRequestWithResponseCallback(request *Message,
	responseCallback ResponseCallback) *requestWithResponseCallback {
	return &requestWithResponseCallback{request: request, responseCallback: responseCallback}
}

type Backend struct {
	addr              string
	stop              int32
	conn              net.Conn
	connected         int32
	responseTimeout   time.Duration
	requests          chan *requestWithResponseCallback
	responseCallbacks *ResponseCallbackMgr
}

// NewBackend create a thrift backend
func NewBackend(addr string) *Backend {
	backend := &Backend{addr: addr,
		stop:              0,
		conn:              NewErrorConn(),
		connected:         0,
		responseTimeout:   60 * time.Second,
		requests:          make(chan *requestWithResponseCallback, 1000),
		responseCallbacks: NewResponseCallbackMgr()}
	go backend.start()
	go backend.startWriteMessage()
	return backend
}

func (b *Backend) GetAddr() string {
	return b.addr
}
func (b *Backend) start() {
	for !b.IsStopped() {
		log.WithFields(log.Fields{"address": b.addr}).Info("try to connect to backend server")
		conn, err := net.Dial("tcp", b.addr)
		if err == nil {
			b.conn = conn
			b.setConnected(true)
			go b.startReadMessage()
			log.WithFields(log.Fields{"address": b.addr}).Info("Connect to backend server successfully")
			break
		} else {
			log.WithFields(log.Fields{"address": b.addr}).Error("Fail to connect backend server")
		}
		time.Sleep(5 * time.Second)
	}
}

func (b *Backend) startReadMessage() {
	buffer := make([]byte, 4096)
	respBuffer := NewMessageBuffer()
	for {
		n, err := b.conn.Read(buffer)
		if err != nil {
			b.setConnected(false)
			log.WithFields(log.Fields{"address": b.addr}).Error("Fail to read response from backend server")
			break
		}
		respBuffer.Add(buffer[0:n])
		b.processResponseBuffer(respBuffer)
	}
	if !b.IsStopped() {
		go b.start()
	}
}

func (b *Backend) isConnected() bool {
	return atomic.LoadInt32(&b.connected) != 0
}

func (b *Backend) setConnected(connected bool) {
	if connected {
		atomic.StoreInt32(&b.connected, 1)
	} else {
		atomic.StoreInt32(&b.connected, 0)
	}
}

// processResponseBuffer process the response from backend server
func (b *Backend) processResponseBuffer(respBuffer *MessageBuffer) {
	for {
		response, err := respBuffer.ExtractMessage()
		if err != nil {
			break
		}
		seqId, err := response.GetSeqId()
		if err == nil {
			respCb, ok := b.getResponseCallback(seqId)
			if ok {
				respCb(response, nil)
			} else {
				log.Error("Fail to find response callback by seqId")
			}
		} else {
			log.Error("Fail to get the seqId from response")
		}
	}
}

func (b *Backend) getResponseCallback(seqId int) (ResponseCallback, bool) {
	return b.responseCallbacks.Remove(seqId)
}

// Stop stop the backend
func (b *Backend) Stop() {
    if atomic.CompareAndSwapInt32(&b.stop, 0, 1)  {
        log.WithFields( log.Fields{ "address": b.addr } ).Info( "Stop backend" )
        defer b.conn.Close()
    } else {
        log.WithFields( log.Fields{ "address": b.addr } ).Info( "Backend is already stopped" )
    }
}

func (b *Backend) IsStopped() bool {
	return atomic.LoadInt32(&b.stop) != 0
}

func (b *Backend) Send(request *Message, callback ResponseCallback) {
	if !b.isConnected() {
		callback(nil, notConnectedError)
	} else {
		b.requests <- newRequestWithResponseCallback(request, callback)
	}
}

func (b *Backend) startWriteMessage() {
	for !b.IsStopped() {
		select {
		case requestWithResponseCb, ok := <-b.requests:
			if !ok {
				log.WithFields(log.Fields{"address": b.addr}).Error("Fail to send request to backend server")
				return
			}
			seqId, _ := requestWithResponseCb.request.GetSeqId()
			b.responseCallbacks.Add(seqId,
				requestWithResponseCb.responseCallback,
				b.responseTimeout)

			err := requestWithResponseCb.request.Write(b.conn)
			if err != nil {
				log.WithFields(log.Fields{"address": b.addr}).Error("Fail to send the request to backend server")
				requestWithResponseCb.responseCallback(nil, err)
				return
			}
		}
	}
}
