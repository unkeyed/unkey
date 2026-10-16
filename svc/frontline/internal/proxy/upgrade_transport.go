package proxy

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptrace"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
	"golang.org/x/net/http/httpguts"
)

// upgradeTransport sends cross-region protocol upgrades over HTTP/1.1. An
// upgrade takes over its connection for the whole session, so it cannot use
// the pooled HTTP/2 connection to the peer frontline.
//
// An upgrade without a body uses a spare connection when one is ready. The
// peer can close a spare while its FIN is still in flight, and http.Transport
// retries only on reused connections, so such an upgrade is retried once on a
// fresh dial. An upgrade with a body cannot be replayed, so it always uses a
// fresh dial.
type upgradeTransport struct {
	spares *http.Transport
	fresh  *http.Transport
}

func newUpgradeTransport(pool *sparePool) *upgradeTransport {
	return &upgradeTransport{
		spares: newHTTP1Transport(pool.dial),
		fresh:  newHTTP1Transport(pool.dialer.DialContext),
	}
}

func newHTTP1Transport(dial func(ctx context.Context, network, addr string) (net.Conn, error)) *http.Transport {
	//nolint:exhaustruct
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialTLSContext:        dial,
		TLSNextProto:          map[string]func(string, *tls.Conn) http.RoundTripper{},
		MaxIdleConnsPerHost:   upgradeSparesPerPeer,
		IdleConnTimeout:       90 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

func newUpgradeDialer() *tls.Dialer {
	//nolint:exhaustruct
	return &tls.Dialer{
		NetDialer: &net.Dialer{
			Timeout:   upgradeSpareDialTime,
			KeepAlive: 30 * time.Second,
		},
		Config: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			ClientSessionCache: tls.NewLRUClientSessionCache(100),
		},
	}
}

func (t *upgradeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.ContentLength != 0 {
		return t.fresh.RoundTrip(req)
	}

	usedSpare := false
	//nolint:exhaustruct
	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			_, usedSpare = info.Conn.(*spareConn)
		},
	}
	resp, err := t.spares.RoundTrip(req.WithContext(httptrace.WithClientTrace(req.Context(), trace)))
	if err == nil || !usedSpare || req.Context().Err() != nil {
		return resp, err
	}

	logger.Warn("upgrade failed on a spare peer connection, retrying on a fresh dial", "target", req.URL.Host, "error", err.Error())
	upgradeSparesTotal.WithLabelValues(spareOutcomeStale).Inc()
	return t.fresh.RoundTrip(req)
}

func isUpgradeRequest(req *http.Request) bool {
	return httpguts.HeaderValuesContainsToken(req.Header["Connection"], "Upgrade") &&
		req.Header.Get("Upgrade") != ""
}
