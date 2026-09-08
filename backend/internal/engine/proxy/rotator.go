package proxy

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Rotator struct {
	proxies []string
	index   int
	mu      sync.Mutex
}

func NewRotator(proxies []string) *Rotator {
	return &Rotator{
		proxies: proxies,
		index:   0,
	}
}

func (r *Rotator) GetNextClient(timeout time.Duration, userAgent string) *http.Client {
	r.mu.Lock()
	defer r.mu.Unlock()

	transport := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives:   false,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	}

	if len(r.proxies) > 0 {
		proxyURLStr := r.proxies[r.index%len(r.proxies)]
		r.index++

		if proxyURL, err := url.Parse(proxyURLStr); err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
}

func (r *Rotator) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.proxies)
}

func (r *Rotator) SetProxies(proxies []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.proxies = proxies
	r.index = 0
}
