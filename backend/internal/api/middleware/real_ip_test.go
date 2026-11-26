package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRealIP(t *testing.T) {
	tests := []struct {
		name     string
		peer     string
		headers  map[string]string
		expected string
	}{
		{"nginx on host", "127.0.0.1:51234", map[string]string{"X-Real-IP": "103.189.63.145"}, "103.189.63.145"},
		{"via docker network", "172.18.0.1:40000", map[string]string{"X-Real-IP": "59.153.130.91"}, "59.153.130.91"},
		{"spoofed True-Client-IP ignored", "127.0.0.1:51234", map[string]string{"X-Real-IP": "103.189.63.145", "True-Client-IP": "1.2.3.4"}, "103.189.63.145"},
		{"spoofed X-Forwarded-For ignored", "127.0.0.1:51234", map[string]string{"X-Forwarded-For": "1.2.3.4, 103.189.63.145"}, "127.0.0.1:51234"},
		{"public peer can't claim another IP", "8.8.8.8:1234", map[string]string{"X-Real-IP": "1.2.3.4"}, "8.8.8.8:1234"},
		{"garbage header ignored", "127.0.0.1:51234", map[string]string{"X-Real-IP": "not-an-ip"}, "127.0.0.1:51234"},
		{"ipv6 client", "[::1]:5000", map[string]string{"X-Real-IP": "2001:db8::1"}, "2001:db8::1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			h := RealIP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { got = r.RemoteAddr }))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.peer
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			h.ServeHTTP(httptest.NewRecorder(), req)
			if got != tt.expected {
				t.Errorf("RemoteAddr = %q, want %q", got, tt.expected)
			}
		})
	}
}
