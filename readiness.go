package main

import (
	"net"
	"net/http"
)

type ReadinessCreator = func(addr string, readinessConf *ReadinessConf) Readiness

type Readiness interface {
	// check if the backend is ready
	IsReady() bool
}

type NullReadiness struct {
}

func NewNullReadiness() *NullReadiness {
	return &NullReadiness{}
}

func (n *NullReadiness) IsReady() bool {
	return true
}

type TcpReadiness struct {
	addr string
}

func NewTcpReadiness(addr string) *TcpReadiness {
	return &TcpReadiness{addr: addr}
}

func (t *TcpReadiness) IsReady() bool {
	conn, err := net.Dial("tcp", t.addr)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

type HttpReadiness struct {
	url string
}

func NewHttpReadiness(url string) *HttpReadiness {
	return &HttpReadiness{url: url}
}

func (h *HttpReadiness) IsReady() bool {
	resp, err := http.Get(h.url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 400

}
