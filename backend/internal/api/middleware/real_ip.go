package middleware

import (
	"net"
	"net/http"
	"strings"
)

// RealIP sets r.RemoteAddr to the client address reported by the reverse proxy.
//
// Only X-Real-IP is used, and only when the request comes straight from a loopback or private
// address (nginx on the host, or another container). nginx sets X-Real-IP to $remote_addr,
// overwriting anything the client sent. Headers that a client can set directly
// (True-Client-IP, the leftmost X-Forwarded-For) are ignored, unlike chi's middleware.RealIP.
func RealIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ip := proxiedClientIP(r); ip != "" {
			r.RemoteAddr = ip
		}
		next.ServeHTTP(w, r)
	})
}

func proxiedClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer := net.ParseIP(host)
	if peer == nil || !(peer.IsLoopback() || peer.IsPrivate()) {
		return "" // not from our proxy: keep the real peer address
	}
	reported := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP")))
	if reported == nil {
		return ""
	}
	return reported.String()
}
