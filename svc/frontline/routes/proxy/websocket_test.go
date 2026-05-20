package handler_test

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
	"github.com/unkeyed/unkey/svc/frontline/internal/errorpage"
	"github.com/unkeyed/unkey/svc/frontline/internal/proxy"
	"github.com/unkeyed/unkey/svc/frontline/internal/router"
	"github.com/unkeyed/unkey/svc/frontline/middleware"
	handler "github.com/unkeyed/unkey/svc/frontline/routes/proxy"
)

const wsAcceptMagic = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// TestWebSocketUpgrade_HTTP1_Succeeds is the regression guard for the WS
// upgrade fix in svc/frontline/internal/proxy/forward.go. Before the fix,
// ModifyResponse wrapped resp.Body in io.NopCloser(io.TeeReader(...)) for
// every response, which destroyed the io.ReadWriteCloser interface that
// httputil.ReverseProxy.handleUpgradeResponse type-asserts when hijacking
// a 101 upstream — yielding "internal error: 101 switching protocols
// response with non-writable body" and a 502 to the client.
func TestWebSocketUpgrade_HTTP1_Succeeds(t *testing.T) {
	t.Parallel()

	backendAddr, stopBackend := startEchoBackend(t)
	t.Cleanup(stopBackend)

	feAddr, stopFrontline := startFrontline(t, backendAddr)
	t.Cleanup(stopFrontline)

	conn, err := net.Dial("tcp", feAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	require.NoError(t, conn.SetDeadline(time.Now().Add(10*time.Second)))

	key := "dGhlIHNhbXBsZSBub25jZQ=="
	require.NoError(t, writeWSHandshake(conn, "ws-test.example.com", "/ws", key))

	br := bufio.NewReader(conn)
	require.NoError(t, expect101(br, key))

	require.NoError(t, writeClientFrame(conn, []byte("hello-ws")))

	payload, err := readServerFrame(br)
	require.NoError(t, err)
	require.Equal(t, "hello-ws", string(payload))
}

// TestWebSocketUpgrade_BackendDown_DoesNotHang verifies the proxy returns
// promptly with a 503 Service Unavailable (the URN ECONNREFUSED maps to
// in proxy/error.go) when the upstream is unreachable, rather than hanging
// or letting the client think it has an upgraded connection.
func TestWebSocketUpgrade_BackendDown_DoesNotHang(t *testing.T) {
	t.Parallel()

	// Listen then close to get a free, *unreachable* address.
	dead, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	deadAddr := dead.Addr().String()
	require.NoError(t, dead.Close())

	feAddr, stopFrontline := startFrontline(t, deadAddr)
	t.Cleanup(stopFrontline)

	conn, err := net.Dial("tcp", feAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	require.NoError(t, conn.SetDeadline(time.Now().Add(10*time.Second)))

	require.NoError(t, writeWSHandshake(conn, "ws-test.example.com", "/ws", "dGhlIHNhbXBsZSBub25jZQ=="))

	br := bufio.NewReader(conn)
	statusLine, err := br.ReadString('\n')
	require.NoError(t, err, "expected a status line, not a hang")
	require.Contains(t, statusLine, " 503 ", "expected 503 Service Unavailable for ECONNREFUSED, got: %q", strings.TrimSpace(statusLine))
}

func startFrontline(t *testing.T, backendAddr string) (string, func()) {
	t.Helper()

	transports := proxy.NewTransportRegistry()
	proxySvc, err := proxy.New(proxy.Config{
		InstanceID:          "test-instance",
		Platform:            "test",
		Region:              "test",
		ApexDomain:          "test.local",
		Clock:               clock.New(),
		MaxHops:             3,
		MaxIdleConns:        0,
		IdleConnTimeout:     0,
		TLSHandshakeTimeout: 0,
		Transport:           nil,
		UpstreamTransports:  transports,
		ErrorPageRenderer:   nil,
	})
	require.NoError(t, err)

	h := &handler.Handler{
		RouterService: &stubRouter{
			decision: router.RouteDecision{
				Destination:      router.DestinationLocalInstance,
				DeploymentID:     "dep_test",
				EnvironmentID:    "env_test",
				WorkspaceID:      "ws_test",
				ProjectID:        "proj_test",
				UpstreamProtocol: db.DeploymentsUpstreamProtocolHttp1,
				Instance: db.FindInstancesByDeploymentIDRow{
					ID:      "inst_test",
					Address: backendAddr,
				},
			},
		},
		ProxyService: proxySvc,
		Engine:       nil,
		Clock:        clock.New(),
	}

	zenSrv, err := zen.New(zen.Config{
		ReadTimeout:        -1,
		WriteTimeout:       -1,
		MaxRequestBodySize: 0,
	})
	require.NoError(t, err)
	// Mirror the production middleware chain in svc/frontline/routes/register.go.
	// Panic recovery + observability are the load-bearing pieces here: without
	// them, a handler error panics in zen and tears the connection — masking
	// what the proxy actually returned.
	mws := []zen.Middleware{
		zen.WithPanicRecovery(),
		middleware.WithReservedHeaderStrip(),
		zen.WithLogging(),
		middleware.WithObservability(errorpage.NewRenderer()),
	}
	zenSrv.RegisterRoute(mws, h)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = zenSrv.Serve(ctx, ln) }()

	waitForListener(t, ln.Addr().String())

	return ln.Addr().String(), func() {
		cancel()
		shutdownCtx, sc := context.WithTimeout(context.Background(), 2*time.Second)
		defer sc()
		_ = zenSrv.Shutdown(shutdownCtx)
	}
}

func waitForListener(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			_ = c.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("listener at %s never became ready", addr)
}

// startEchoBackend stands up a stdlib WS echo server. It performs the
// RFC 6455 handshake by hand via http.Hijacker and echoes one short text
// frame, then closes. Using stdlib (rather than a 3rd-party WS lib) keeps
// go.mod clean and avoids hiding behavior that might mask a frontline bug.
func startEchoBackend(t *testing.T) (string, func()) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := &http.Server{
		ReadTimeout:  0,
		WriteTimeout: 0,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
				http.Error(w, "expected websocket upgrade", http.StatusBadRequest)
				return
			}
			key := r.Header.Get("Sec-WebSocket-Key")
			if key == "" {
				http.Error(w, "missing Sec-WebSocket-Key", http.StatusBadRequest)
				return
			}

			hj, ok := w.(http.Hijacker)
			if !ok {
				http.Error(w, "hijack unsupported", http.StatusInternalServerError)
				return
			}
			conn, brw, err := hj.Hijack()
			if err != nil {
				return
			}
			defer func() { _ = conn.Close() }()

			resp := "HTTP/1.1 101 Switching Protocols\r\n" +
				"Upgrade: websocket\r\n" +
				"Connection: Upgrade\r\n" +
				"Sec-WebSocket-Accept: " + wsAccept(key) + "\r\n\r\n"
			if _, err := brw.WriteString(resp); err != nil {
				return
			}
			if err := brw.Flush(); err != nil {
				return
			}

			payload, err := readClientFrame(brw.Reader)
			if err != nil {
				return
			}
			_ = writeServerFrame(conn, payload)
		}),
	}

	go func() { _ = srv.Serve(ln) }()

	return ln.Addr().String(), func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}
}

type stubRouter struct {
	decision router.RouteDecision
}

func (s *stubRouter) Route(_ context.Context, _ string) (router.RouteDecision, error) {
	return s.decision, nil
}

func (s *stubRouter) ValidateHostname(_ context.Context, _ string) error { return nil }

// --- Hand-rolled WS framing ---

func wsAccept(key string) string {
	h := sha1.New()
	h.Write([]byte(key))
	h.Write([]byte(wsAcceptMagic))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func writeWSHandshake(w io.Writer, host, path, key string) error {
	req := "GET " + path + " HTTP/1.1\r\n" +
		"Host: " + host + "\r\n" +
		"Connection: Upgrade\r\n" +
		"Upgrade: websocket\r\n" +
		"Sec-WebSocket-Version: 13\r\n" +
		"Sec-WebSocket-Key: " + key + "\r\n\r\n"
	_, err := io.WriteString(w, req)
	return err
}

func expect101(br *bufio.Reader, sentKey string) error {
	statusLine, err := br.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read status line: %w", err)
	}
	if !strings.Contains(statusLine, " 101 ") {
		var rest strings.Builder
		_, _ = io.Copy(&rest, br)
		return fmt.Errorf("expected 101 Switching Protocols, got: %q (rest: %q)", strings.TrimSpace(statusLine), rest.String())
	}
	headers := http.Header{}
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read header line: %w", err)
		}
		if line == "\r\n" || line == "\n" {
			break
		}
		idx := strings.IndexByte(line, ':')
		if idx < 0 {
			continue
		}
		k := strings.TrimSpace(line[:idx])
		v := strings.TrimSpace(line[idx+1:])
		headers.Add(k, v)
	}
	if got, want := headers.Get("Sec-WebSocket-Accept"), wsAccept(sentKey); got != want {
		return fmt.Errorf("Sec-WebSocket-Accept mismatch: got %q want %q", got, want)
	}
	if !strings.EqualFold(headers.Get("Upgrade"), "websocket") {
		return fmt.Errorf("Upgrade header missing or wrong: %q", headers.Get("Upgrade"))
	}
	return nil
}

// writeClientFrame writes a single masked text frame. Length must fit in 7 bits.
func writeClientFrame(w io.Writer, payload []byte) error {
	if len(payload) >= 126 {
		return fmt.Errorf("test helper only supports payloads <126 bytes, got %d", len(payload))
	}
	hdr := []byte{0x81, byte(len(payload)) | 0x80}
	if _, err := w.Write(hdr); err != nil {
		return err
	}
	var mask [4]byte
	if _, err := rand.Read(mask[:]); err != nil {
		return err
	}
	if _, err := w.Write(mask[:]); err != nil {
		return err
	}
	masked := make([]byte, len(payload))
	for i := range payload {
		masked[i] = payload[i] ^ mask[i%4]
	}
	_, err := w.Write(masked)
	return err
}

// readClientFrame reads one short masked text frame and returns the unmasked payload.
func readClientFrame(r *bufio.Reader) ([]byte, error) {
	hdr := make([]byte, 2)
	if _, err := io.ReadFull(r, hdr); err != nil {
		return nil, err
	}
	if op := hdr[0] & 0x0f; op != 0x1 {
		return nil, fmt.Errorf("expected text frame, got opcode %d", op)
	}
	masked := hdr[1]&0x80 != 0
	plen := int(hdr[1] & 0x7f)
	if plen >= 126 {
		return nil, fmt.Errorf("test helper only supports payloads <126 bytes")
	}
	var mask [4]byte
	if masked {
		if _, err := io.ReadFull(r, mask[:]); err != nil {
			return nil, err
		}
	}
	payload := make([]byte, plen)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return payload, nil
}

// readServerFrame reads one short unmasked text frame from the server.
func readServerFrame(r *bufio.Reader) ([]byte, error) {
	hdr := make([]byte, 2)
	if _, err := io.ReadFull(r, hdr); err != nil {
		return nil, err
	}
	if op := hdr[0] & 0x0f; op != 0x1 {
		return nil, fmt.Errorf("expected text frame, got opcode %d", op)
	}
	if hdr[1]&0x80 != 0 {
		return nil, fmt.Errorf("server frames must not be masked")
	}
	plen := int(hdr[1] & 0x7f)
	if plen >= 126 {
		return nil, fmt.Errorf("test helper only supports payloads <126 bytes")
	}
	payload := make([]byte, plen)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// writeServerFrame writes one short unmasked text frame.
func writeServerFrame(w io.Writer, payload []byte) error {
	if len(payload) >= 126 {
		return fmt.Errorf("test helper only supports payloads <126 bytes")
	}
	hdr := []byte{0x81, byte(len(payload))}
	if _, err := w.Write(hdr); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}
