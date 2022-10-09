package pkg

import (
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

var (
	localDockerIpRegex = regexp.MustCompile(`^172\.\d{1,3}\.0\.1:\d{1,5}`)
)

func IPIsLocal(ipAddr string) bool {
	// used in local development ?
	if strings.HasPrefix(ipAddr, "127.0.0.1:") {
		return true
	}

	// user within docker container ?
	return localDockerIpRegex.MatchString(ipAddr)
}

func ReadUserIP(r *http.Request) (string, error) {
	ipAddr := r.Header.Get("X-Real-Ip")
	if ipAddr == "" {
		ipAddr = r.Header.Get("X-Forwarded-For")
	}
	if ipAddr == "" {
		ipAddr = r.RemoteAddr
	}

	// used in development
	if IPIsLocal(ipAddr) {
		log.Debugf("read user IP: returning development localhost / Berlin")
		return "localhost", nil
	}

	ip := net.ParseIP(ipAddr)
	if ip == nil {
		return "", fmt.Errorf("ip addr %s is invalid", ipAddr)
	}

	if strings.Contains(ipAddr, ":") {
		ipAddr = strings.Split(ipAddr, ":")[0]
	}

	return ipAddr, nil
}
