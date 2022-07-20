package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type IPResolvedCallback = func(hostname string, newIPs []string, removedIPs []string)

type addressWithCallback struct {
	addrs      map[string]time.Time
	addrExpire time.Duration
	callback   IPResolvedCallback
}

// Resolver dynamically resolve the host name to IP addresses
type Resolver struct {
	sync.Mutex
	//resolve interval
	interval time.Duration

	// 0: no stop, 1: stop the resolve
	stop int32

	hostIPs map[string]*addressWithCallback
}

var ADDRESS_EXPIRE time.Duration = time.Duration(60)

func init() {
	expire := os.Getenv("ADDRESS_EXPIRE")
	if len(expire) <= 0 {
		expire = "60s"
	}
	d, err := time.ParseDuration(expire)
	if err != nil {
		d = time.Duration(60)
	}
	ADDRESS_EXPIRE = d

}

func newAddressWithCallback(callback IPResolvedCallback, addrExpire time.Duration) *addressWithCallback {
	return &addressWithCallback{addrs: make(map[string]time.Time),
		addrExpire: addrExpire,
		callback:   callback}
}

func (ac *addressWithCallback) addAddrs(addrs []string) {
	for _, addr := range addrs {
		ac.addrs[addr] = time.Now().Add(ac.addrExpire)
	}
}

func (ac *addressWithCallback) cleanExpiredAddrs() []string {
	expiredAddrs := make([]string, 0)
	for addr, expireInfo := range ac.addrs {
		if expireInfo.Before(time.Now()) {
			expiredAddrs = append(expiredAddrs, addr)
		}
	}

	for _, addr := range expiredAddrs {
		delete(ac.addrs, addr)
	}
	return expiredAddrs
}

func (ac *addressWithCallback) getAddrs() []string {
	addrs := make([]string, 0)
	for addr := range ac.addrs {
		addrs = append(addrs, addr)
	}
	return addrs

}
func NewResolver(interval int) *Resolver {
	r := &Resolver{interval: time.Duration(interval) * time.Second,
		stop:    0,
		hostIPs: make(map[string]*addressWithCallback)}
	go r.periodicalResolve()
	return r
}

func (r *Resolver) ResolveHost(addr string, callback IPResolvedCallback) {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.hostIPs[addr]; !ok {
		r.hostIPs[addr] = newAddressWithCallback(callback, ADDRESS_EXPIRE)
		ips, err := r.doResolve(addr)
		if err == nil {
			r.hostIPs[addr].addAddrs(ips)
			callback(addr, ips, make([]string, 0))
		}
	}
}

func (r *Resolver) StopResolve(hostname string) {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.hostIPs[hostname]; ok {
		delete(r.hostIPs, hostname)

	}
}

func (r *Resolver) getHostnames() []string {
	r.Lock()
	defer r.Unlock()

	hostnames := make([]string, 0)

	for hostname, _ := range r.hostIPs {
		hostnames = append(hostnames, hostname)
	}
	return hostnames
}

// Stop stop the hostname resolve
func (r *Resolver) Stop() {
	if atomic.CompareAndSwapInt32(&r.stop, 0, 1) {
		log.Info("stop the hostname to IP resolve")
	}
}

func (r *Resolver) isStopped() bool {
	return atomic.LoadInt32(&r.stop) != 0
}

func (r *Resolver) GetAddrsOfHost(hostname string) []string {
	r.Lock()
	defer r.Unlock()

	if v, ok := r.hostIPs[hostname]; ok {
		return v.getAddrs()
	}
	return make([]string, 0)
}
func (r *Resolver) periodicalResolve() {
	for !r.isStopped() {
		hostnames := r.getHostnames()

		for _, hostname := range hostnames {
			addrs, err := r.doResolve(hostname)
			if err != nil {
				log.WithFields(log.Fields{"hostname": hostname}).Error("Fail to resolve host to IP")
			}
			r.addressResolved(hostname, addrs, err)
		}
		time.Sleep(r.interval)
	}
}

func (r *Resolver) addressResolved(hostname string, addrs []string, err error) {
	r.Lock()
	defer r.Unlock()
	if entry, ok := r.hostIPs[hostname]; ok {
		if err == nil {
			removedAddrs := entry.cleanExpiredAddrs()
			oldAddrs := entry.getAddrs()
			entry.addAddrs(addrs)
			newAddrs := strArraySub(addrs, oldAddrs)
			if len(newAddrs) > 0 || len(removedAddrs) > 0 {
				log.WithFields(log.Fields{"hostname": hostname, "newAddrs": strings.Join(newAddrs, ","), "removedAddrs": strings.Join(removedAddrs, ",")}).Info("the ip address of host is changed")
				go entry.callback(hostname, newAddrs, removedAddrs)
			}
		}
	}
}

func (r *Resolver) doResolve(addr string) ([]string, error) {
	hostname, port, err := splitAddr(addr)

	if err != nil {
		return nil, err
	}

	ips, err := net.LookupIP(hostname)

	if err != nil {
		return nil, err
	}

	result := make([]string, 0)
	for _, ip := range ips {
		s := ip.String()
		if strings.Index(s, ":") != -1 {
			s = fmt.Sprintf("[%s]", s)
		}
		result = append(result, fmt.Sprintf("%s:%s", s, port))
	}
	return result, nil
}
