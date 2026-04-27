package proxy

import (
	"bufio"
	"crypto/tls"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
)

type Handler struct {
	certCache       *CertCache
	transport       *http.Transport
	outpostRequests *batch.BatchProcessor[schema.OutpostRequest]
	outpostID       string
	region          string
}

func NewHandler(certCache *CertCache, transport *http.Transport, outpostRequests *batch.BatchProcessor[schema.OutpostRequest], outpostID string, region string) *Handler {
	return &Handler{
		certCache:       certCache,
		transport:       transport,
		outpostRequests: outpostRequests,
		outpostID:       outpostID,
		region:          region,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodConnect {
		http.Error(w, "only HTTPS egress is allowed", http.StatusMethodNotAllowed)
		return
	}

	h.handleConnect(w, r)
}

func (h *Handler) handleConnect(w http.ResponseWriter, r *http.Request) {
	host := extractHost(r.Host)
	identity := ParseIdentity(r)
	if err := assert.All(
		assert.NotEmpty(identity.WorkspaceID, "proxy identity missing workspace_id"),
		assert.NotEmpty(identity.DeploymentID, "proxy identity missing deployment_id"),
	); err != nil {
		logger.Error("krane setup is broken", "host", host, "error", err)
		http.Error(w, "missing proxy identity", http.StatusProxyAuthRequired)
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijack not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		logger.Error("hijack failed", "host", host, "error", err)
		return
	}
	defer func() { _ = clientConn.Close() }()

	if _, err := clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		logger.Error("connect response failed", "host", host, "error", err)
		return
	}

	leafCert, err := h.certCache.GetOrCreate(host)
	if err != nil {
		logger.Error("cert generation failed", "host", host, "error", err)
		return
	}

	//nolint:exhaustruct
	tlsConn := tls.Server(clientConn, &tls.Config{
		Certificates: []tls.Certificate{*leafCert},
		MinVersion:   tls.VersionTLS12,
	})
	defer func() { _ = tlsConn.Close() }()

	if err := tlsConn.Handshake(); err != nil {
		logger.Error("mitm handshake failed", "host", host, "error", err)
		return
	}

	clientReader := bufio.NewReader(tlsConn)

	for {
		innerReq, err := http.ReadRequest(clientReader)
		if err != nil {
			if err != io.EOF {
				logger.Error("read inner request failed", "host", host, "error", err)
			}
			return
		}

		sentinelRequestID := innerReq.Header.Get("X-Unkey-Request-Id")

		innerReq.URL.Scheme = "https"
		innerReq.URL.Host = r.Host
		innerReq.RequestURI = ""
		innerReq.Header.Del("Proxy-Authorization")
		innerReq.Header.Del("Proxy-Connection")
		innerReq.Header.Del("X-Unkey-Request-Id")
		innerReq.Header.Set("X-Outpost-Forwarded", "true")

		start := time.Now()
		resp, err := h.transport.RoundTrip(innerReq)
		latency := time.Since(start)

		if err != nil {
			logger.Error("upstream request failed", "host", host, "error", err)
			h.outpostRequests.Buffer(schema.OutpostRequest{
				RequestID:         uid.New(uid.RequestPrefix),
				Time:              start.UnixMilli(),
				OutpostID:         h.outpostID,
				SentinelRequestID: sentinelRequestID,
				WorkspaceID:       identity.WorkspaceID,
				DeploymentID:      identity.DeploymentID,
				Region:            h.region,
				DestinationHost:   host,
				Method:            innerReq.Method,
				Path:              innerReq.URL.Path,
				LatencyMs:         latency.Milliseconds(),
				RequestBytes:      0,
				ResponseBytes:     0,
				ResponseStatus:    0,
				Error:             err.Error(),
			})
			writeHTTPError(tlsConn, http.StatusBadGateway)
			return
		}

		statusCode := resp.StatusCode
		contentLength := resp.ContentLength
		requestLength := innerReq.ContentLength
		method := innerReq.Method
		path := innerReq.URL.Path

		writeErr := resp.Write(tlsConn)
		_ = resp.Body.Close()

		logger.Info("outpost",
			slog.String("host", host),
			slog.String("method", method),
			slog.String("path", path),
			slog.Int("status", statusCode),
			slog.Duration("latency", latency),
		)

		errStr := ""
		if writeErr != nil {
			errStr = writeErr.Error()
		}

		h.outpostRequests.Buffer(schema.OutpostRequest{
			RequestID:         uid.New(uid.RequestPrefix),
			Time:              start.UnixMilli(),
			OutpostID:         h.outpostID,
			SentinelRequestID: sentinelRequestID,
			WorkspaceID:       identity.WorkspaceID,
			DeploymentID:      identity.DeploymentID,
			Region:            h.region,
			DestinationHost:   host,
			Method:            method,
			Path:              path,
			ResponseStatus:    int32(statusCode),
			LatencyMs:         latency.Milliseconds(),
			ResponseBytes:     contentLength,
			RequestBytes:      requestLength,
			Error:             errStr,
		})

		if writeErr != nil {
			return
		}
	}
}

func writeHTTPError(conn net.Conn, status int) {
	//nolint:exhaustruct
	resp := &http.Response{
		StatusCode: status,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
	}
	_ = resp.Write(conn)
}

func extractHost(hostport string) string {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport
	}
	return host
}
