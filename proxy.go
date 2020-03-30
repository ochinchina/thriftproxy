package main

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
)

type ProxyMgr struct {
	proxies []*Proxy
}

func NewProxyMgr() *ProxyMgr {
	return &ProxyMgr{proxies: make([]*Proxy, 0)}
}

func (p *ProxyMgr) AddProxy(proxy *Proxy) {
	p.proxies = append(p.proxies, proxy)
}

func (p *ProxyMgr) RemoveProxy(name string) {
	for index, proxy := range p.proxies {
		if proxy.name == name {
			p.proxies = append(p.proxies[0:index], p.proxies[index+1:]...)
		}
	}
}

func (p *ProxyMgr) GetProxy(name string) (*Proxy, error) {
	for _, proxy := range p.proxies {
		if proxy.name == name {
			return proxy, nil
		}
	}
	return nil, errors.New("Fail to find proxy")
}

// Start start all the proxies
func (p *ProxyMgr) Run() {
	var wg sync.WaitGroup
	for _, proxy := range p.proxies {
		wg.Add(1)
		go p.startProxy(proxy, &wg)
	}

	wg.Wait()
}

func (p *ProxyMgr) startProxy(proxy *Proxy, wg *sync.WaitGroup) {
	defer wg.Done()
	proxy.Run()
}

type Proxy struct {
	name           string
	addr           string
	seqIdAllocator *SeqIdAllocator
	loadBalancer   LoadBalancer
	clients        []*Client
	clientLock     sync.Mutex
}

// NewProxy create a thrift proxy listening on the addr
// and all received message will be forward by loadBalancer
// to backend thrift servers
func NewProxy(name string,
	addr string,
	loadBalancer LoadBalancer) *Proxy {
	proxy := &Proxy{name: name,
		addr:           addr,
		seqIdAllocator: NewSeqIdAllocator(),
		loadBalancer:   loadBalancer,
		clients:        make([]*Client, 0)}

	return proxy
}

func (p *Proxy) Run() {
	log.WithFields(log.Fields{"name": p.name}).Info("Start proxy")
	ln, err := net.Listen("tcp", p.addr)
	if err != nil {
		log.WithFields(log.Fields{"address": p.addr}).Error("Fail to listen on address")
		return
	}

	log.WithFields(log.Fields{"address": p.addr}).Info("Listen on address")

	for {
		conn, err := ln.Accept()
		if err == nil {
			client := NewClient(conn,
				p.seqIdAllocator,
				p.loadBalancer,
				p.removeClient)

			log.WithFields(log.Fields{"address": conn.RemoteAddr().String()}).Info("Accept connection")
			p.addClient(client)
		}
	}
}

func (p *Proxy) AddBackend(backendInfo *BackendInfo) {
	p.loadBalancer.AddBackend(backendInfo)
}

func (p *Proxy) RemoveBackend(addr string) {
	p.loadBalancer.RemoveBackend(addr)
}

func (p *Proxy) addClient(client *Client) {
	p.clientLock.Lock()
	defer p.clientLock.Unlock()
	p.clients = append(p.clients, client)

}

func (p *Proxy) removeClient(c *Client) {
	p.clientLock.Lock()
	defer p.clientLock.Unlock()

	for i, value := range p.clients {
		if value == c {
			log.WithFields(log.Fields{"address": c.remoteAddr().String()}).Info("Succeed to remove client")
			p.clients = append(p.clients[:i], p.clients[i+1:]...)
			break
		}
	}
}
