package proxy

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

func NewOutboundTransport() *http.Transport {
	//nolint:exhaustruct
	return &http.Transport{
		ForceAttemptHTTP2: true,
		DialContext: (&net.Dialer{ //nolint:exhaustruct
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		//nolint:exhaustruct
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			ClientSessionCache: tls.NewLRUClientSessionCache(100),
		},
	}
}
