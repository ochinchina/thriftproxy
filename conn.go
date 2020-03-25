package main

import (
	"net"
	"time"
)

var noAddr *NoAddr = NewNoAddr()

type ErrorConn struct {
}

type NoAddr struct {
}

func NewNoAddr() *NoAddr {
	return &NoAddr{}
}

func (n *NoAddr) Network() string {
	return "No network"
}

func (n *NoAddr) String() string {
	return "No addr"
}

func NewErrorConn() *ErrorConn {
	return &ErrorConn{}
}

func (e *ErrorConn) Read(b []byte) (int, error) {
	return 0, notConnectedError
}

func (e *ErrorConn) Write(b []byte) (int, error) {
	return 0, notConnectedError
}

func (e *ErrorConn) Close() error {
	return notConnectedError
}

func (e *ErrorConn) LocalAddr() net.Addr {
	return noAddr
}

func (e *ErrorConn) RemoteAddr() net.Addr {
	return noAddr
}

func (e *ErrorConn) SetDeadline(t time.Time) error {
	return notConnectedError
}

func (e *ErrorConn) SetReadDeadline(t time.Time) error {
	return notConnectedError
}

func (e *ErrorConn) SetWriteDeadline(t time.Time) error {
	return notConnectedError
}
