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

// upgradeSpareResponseTimeout bounds the wait for a response on a spare. A
// peer pod that is gone without a FIN or RST leaves a spare that accepts the
// request but never answers.
const upgradeSpareResponseTimeout = 10 * time.Second

// upgradeTransport sends cross-region protocol upgrades over HTTP/1.1. An
// upgrade takes over its connection for the whole session, so it cannot use
// the pooled HTTP/2 connection to the peer frontline.
//
// A GET upgrade without a body uses a spare connection when one is ready. If
// the spare fails before a response arrives, for example because the peer
// closed it while its FIN was still in flight, the upgrade is retried once on
// a fresh dial. Other upgrades always use a fresh dial.
type upgradeTransport struct {
	spares *http.Transport
	fresh  *http.Transport
}

type upgradeTransportConfig struct {
	maxIdleConns         int
	idleConnTimeout      time.Duration
	spareResponseTimeout time.Duration
}

func newUpgradeTransport(pool *sparePool, cfg upgradeTransportConfig) *upgradeTransport {
	spares := newHTTP1Transport(pool.dial, cfg)
	spares.ResponseHeaderTimeout = cfg.spareResponseTimeout
	return &upgradeTransport{
		spares: spares,
		fresh:  newHTTP1Transport(pool.dialer.DialContext, cfg),
	}
}

func newHTTP1Transport(dial func(ctx context.Context, network, addr string) (net.Conn, error), cfg upgradeTransportConfig) *http.Transport {
	//nolint:exhaustruct
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialTLSContext:        dial,
		TLSNextProto:          map[string]func(string, *tls.Conn) http.RoundTripper{},
		MaxIdleConns:          cfg.maxIdleConns,
		MaxIdleConnsPerHost:   upgradeSparesPerPeer,
		IdleConnTimeout:       cfg.idleConnTimeout,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// newUpgradeDialer returns the dialer for peer upgrade connections. Its
// timeout covers the TCP connect and the TLS handshake together.
func newUpgradeDialer(tlsHandshakeTimeout time.Duration) *tls.Dialer {
	//nolint:exhaustruct
	return &tls.Dialer{
		NetDialer: &net.Dialer{
			Timeout:   10*time.Second + tlsHandshakeTimeout,
			KeepAlive: 30 * time.Second,
		},
		Config: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			ClientSessionCache: tls.NewLRUClientSessionCache(100),
		},
	}
}

func (t *upgradeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet || req.ContentLength != 0 {
		upgradeSparesTotal.WithLabelValues(spareOutcomeSkipped).Inc()
		return t.fresh.RoundTrip(req)
	}

	var conn httptrace.GotConnInfo
	//nolint:exhaustruct
	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) { conn = info },
	}
	resp, err := t.spares.RoundTrip(req.WithContext(httptrace.WithClientTrace(req.Context(), trace)))
	_, usedSpare := conn.Conn.(*spareConn)

	switch {
	case conn.Reused:
		upgradeSparesTotal.WithLabelValues(spareOutcomeReused).Inc()
		return resp, err
	case !usedSpare:
		upgradeSparesTotal.WithLabelValues(spareOutcomeMiss).Inc()
		return resp, err
	case err == nil:
		upgradeSparesTotal.WithLabelValues(spareOutcomeHit).Inc()
		return resp, err
	}

	upgradeSparesTotal.WithLabelValues(spareOutcomeStale).Inc()
	if req.Context().Err() != nil {
		return resp, err
	}

	// This retry is a deliberate exception to the rule that proxy retries
	// happen only on dial failures. The request was written, but it is a GET
	// without a body and no response arrived, so a replay can at most repeat
	// an upstream handshake that the client never saw.
	logger.Warn("upgrade failed on a spare peer connection, retrying on a fresh dial", "target", req.URL.Host, "error", err.Error())
	return t.fresh.RoundTrip(req)
}

func isUpgradeRequest(req *http.Request) bool {
	return httpguts.HeaderValuesContainsToken(req.Header["Connection"], "Upgrade") &&
		req.Header.Get("Upgrade") != ""
}
