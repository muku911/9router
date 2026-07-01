package utils

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"sync/atomic"
)

// ProxyPoolRoundTripper is a custom http.RoundTripper that rotates proxies
// in a round-robin fashion for each outgoing request.
type ProxyPoolRoundTripper struct {
	proxies     []*url.URL
	counter     uint32
	baseTrans   *http.Transport
	strictProxy bool
}

func NewProxyPoolRoundTripper(proxyURLs []string, strictProxy bool) (*ProxyPoolRoundTripper, error) {
	var parsedProxies []*url.URL
	for _, pStr := range proxyURLs {
		if pStr == "" {
			continue
		}
		pURL, err := url.Parse(pStr)
		if err != nil {
			return nil, err
		}
		parsedProxies = append(parsedProxies, pURL)
	}

	baseTrans := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
	}

	return &ProxyPoolRoundTripper{
		proxies:     parsedProxies,
		baseTrans:   baseTrans,
		strictProxy: strictProxy,
	}, nil
}

func (rt *ProxyPoolRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if len(rt.proxies) == 0 {
		// No proxies in the pool, fallback to direct request
		return rt.baseTrans.RoundTrip(req)
	}

	// Rotate proxy using atomic increment
	idx := atomic.AddUint32(&rt.counter, 1) % uint32(len(rt.proxies))
	proxy := rt.proxies[idx]

	// Clone the transport to change Proxy safely without race conditions
	clonedTransport := rt.baseTrans.Clone()
	clonedTransport.Proxy = http.ProxyURL(proxy)

	resp, err := clonedTransport.RoundTrip(req)
	if err != nil && rt.strictProxy {
		return nil, err
	} else if err != nil {
		// Fallback to direct routing if strictProxy is false
		return rt.baseTrans.RoundTrip(req)
	}

	return resp, nil
}
