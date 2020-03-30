package main

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

func inStrArray(s string, a []string) bool {
	for _, t := range a {
		if t == s {
			return true
		}
	}
	return false
}
func strArraySub(a1 []string, a2 []string) []string {
	r := make([]string, 0)
	for _, s := range a1 {
		if !inStrArray(s, a2) {
			r = append(r, s)
		}
	}
	return r
}

func isIPAddress(addr string) bool {
	if strings.HasPrefix(addr, "[") && strings.HasSuffix(addr, "]") {
		addr = addr[1 : len(addr)-1]
	}
	return net.ParseIP(addr) != nil
}

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

func createReadiness(addr string, readinessConf *ReadinessConf) Readiness {
	if readinessConf == nil {
		return NewNullReadiness()
	}
	ip, _, err := splitAddr(addr)
	if err != nil {
		ip = addr
	}
	//if it is IPv6
	if strings.Index(ip, ":") != -1 && !strings.HasPrefix(ip, "[") {
		ip = fmt.Sprintf("[%s]", ip)
	}
	switch readinessConf.Protocol {
	case "tcp":
		return NewTcpReadiness(fmt.Sprintf("%s:%d", ip, readinessConf.Port))
	case "http":
		path := "/"
		if len(readinessConf.Path) > 0 {
			path = readinessConf.Path
		}
		url := fmt.Sprintf("http://%s:%d%s", ip, readinessConf.Port, path)
		return NewHttpReadiness(url)
	default:
		return NewNullReadiness()
	}
}

func convertDuration(duration string, defDuration time.Duration) time.Duration {
	n := len(duration)
	if n <= 0 {
		return defDuration
	}
	if strings.HasSuffix(duration, "ms") {
		t, err := strconv.Atoi(duration[0 : n-2])
		if err == nil {
			return time.Duration(t) * time.Millisecond
		}
	} else if strings.HasSuffix(duration, "s") {
		t, err := strconv.Atoi(duration[0 : n-1])
		if err == nil {
			return time.Duration(t) * time.Second
		}
	}
	return defDuration
}
