package api

import (
	"net"
	"net/http"
	"regexp"
)

func isValidHostname(host string) (string, bool) {
	valid, _ := regexp.Match("^([a-zA-Z0-9]([a-zA-Z0-9\\-]{0,61}[a-zA-Z0-9])?)$", []byte(host))
	return host, valid
}

func extractRemoteAddr(req *http.Request) (string, error) {
	header_data, ok := req.Header["X-Forwarded-For"]

	if ok {
		return header_data[0], nil
	} else {
		ip, _, err := net.SplitHostPort(req.RemoteAddr)
		return ip, err
	}
}
