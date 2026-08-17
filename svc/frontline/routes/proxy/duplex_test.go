package handler_test

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/unkeyed/unkey/svc/frontline/internal/db"
	"github.com/unkeyed/unkey/svc/frontline/internal/proxy"
)

func TestH2CFullDuplex_HTTP1Ingress(t *testing.T) {
	t.Parallel()
	testH2CFullDuplex(t, false, http.DefaultClient)
}

func TestH2CFullDuplex_HTTP2Ingress(t *testing.T) {
	t.Parallel()
	testH2CFullDuplex(t, true, &http.Client{
		Transport: proxy.NewTransportRegistry().Get(db.DeploymentsUpstreamProtocolH2c),
	})
}

func testH2CFullDuplex(t *testing.T, enableH2CIngress bool, client *http.Client) {
	t.Helper()

	backendAddr, stopBackend := startH2CDuplexBackend(t)
	t.Cleanup(stopBackend)

	decision := localDecision(backendAddr)
	decision.UpstreamProtocol = db.DeploymentsUpstreamProtocolH2c
	frontlineAddr, stopFrontline := startFrontlineWithH2CIngress(t, decision, nil, enableH2CIngress)
	t.Cleanup(stopFrontline)

	requestBody, requestWriter := io.Pipe()
	t.Cleanup(func() { _ = requestWriter.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+frontlineAddr+"/duplex", requestBody)
	require.NoError(t, err)
	req.Host = "duplex-test.example.com"

	type responseResult struct {
		response *http.Response
		err      error
	}
	responseCh := make(chan responseResult, 1)
	go func() {
		resp, doErr := client.Do(req)
		responseCh <- responseResult{response: resp, err: doErr}
	}()

	_, err = io.WriteString(requestWriter, "first\n")
	require.NoError(t, err)

	var result responseResult
	select {
	case result = <-responseCh:
	case <-time.After(2 * time.Second):
		_ = requestWriter.Close()
		t.Fatal("frontline did not return the upstream response while the request body remained open")
	}
	require.NoError(t, result.err)
	t.Cleanup(func() { _ = result.response.Body.Close() })
	require.Equal(t, http.StatusOK, result.response.StatusCode)

	responseBody := bufio.NewReader(result.response.Body)
	first, err := responseBody.ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "ack:first\n", first)

	_, err = io.WriteString(requestWriter, "second\n")
	require.NoError(t, err)
	require.NoError(t, requestWriter.Close())

	second, err := responseBody.ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "ack:second\n", second)
}

func startH2CDuplexBackend(t *testing.T) (string, func()) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "response writer does not support flushing", http.StatusInternalServerError)
			return
		}

		body := bufio.NewReader(r.Body)
		for range 2 {
			line, readErr := body.ReadString('\n')
			if readErr != nil {
				http.Error(w, "failed to read request stream", http.StatusBadRequest)
				return
			}
			if _, writeErr := io.WriteString(w, "ack:"+line); writeErr != nil {
				return
			}
			flusher.Flush()
		}
	})

	//nolint:exhaustruct
	srv := &http.Server{Handler: h2c.NewHandler(handler, &http2.Server{})}
	go func() { _ = srv.Serve(ln) }()

	return ln.Addr().String(), func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}
}
