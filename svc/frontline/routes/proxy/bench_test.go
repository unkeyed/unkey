package handler_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/internal/services/keys"
	rl "github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/errorpage"
	"github.com/unkeyed/unkey/svc/frontline/internal/policies"
	"github.com/unkeyed/unkey/svc/frontline/internal/proxy"
	"github.com/unkeyed/unkey/svc/frontline/middleware"
	handler "github.com/unkeyed/unkey/svc/frontline/routes/proxy"
)

// Benchmarks for the request-logging overhead on the local-instance proxy
// path. Each benchmark drives real HTTP over loopback through the production
// middleware chain (panic recovery, header strip, ClickHouse logging,
// observability) into the proxy handler and a real upstream backend.
//
// The ClickHouse batch flush is a no-op, so the numbers include row
// construction and buffering but not ClickHouse network I/O. zen.WithLogging
// is omitted to keep benchmark output readable; it is identical across
// variants and does not affect the comparison.
//
// Run with:
//
//	mise exec -- go test -bench BenchmarkProxy -benchmem -run xxx ./svc/frontline/routes/proxy
//
// Compare against main (always-on logging) by running the equivalent
// benchmark there.

// benchKeyService and benchRatelimiter satisfy the engine constructor. A
// logging-only policy set never invokes them.
type benchKeyService struct{}

func (benchKeyService) Get(context.Context, *zen.Session, string) (*keys.KeyVerifier, error) {
	panic("keyauth is not exercised by this benchmark")
}

func (benchKeyService) GetRootKey(context.Context, *zen.Session) (*keys.KeyVerifier, error) {
	panic("keyauth is not exercised by this benchmark")
}

func (benchKeyService) GetMigrated(context.Context, *zen.Session, string, string) (*keys.KeyVerifier, error) {
	panic("keyauth is not exercised by this benchmark")
}

func (benchKeyService) CreateKey(context.Context, keys.CreateKeyRequest) (keys.CreateKeyResponse, error) {
	panic("keyauth is not exercised by this benchmark")
}

type benchRatelimiter struct{}

func (benchRatelimiter) Ratelimit(context.Context, rl.RatelimitRequest) (rl.RatelimitResponse, error) {
	panic("ratelimit is not exercised by this benchmark")
}

func (benchRatelimiter) RatelimitMany(context.Context, []rl.RatelimitRequest) ([]rl.RatelimitResponse, error) {
	panic("ratelimit is not exercised by this benchmark")
}

func loggingPolicy() *frontlinev1.Policy {
	//nolint:exhaustruct
	return &frontlinev1.Policy{
		Id:      "pol_bench_logging",
		Name:    "bench logging",
		Enabled: proto.Bool(true),
		Config:  &frontlinev1.Policy_Logging{Logging: &frontlinev1.Logging{}},
	}
}

// startBenchFrontline boots a frontline server whose router always resolves
// to the given backend with the given policies, mirroring the production
// middleware chain from routes/register.go. rowsBuffered counts ClickHouse
// rows that reached the batch flush, so benchmarks can assert the logging
// path is (in)active rather than silently measuring the wrong thing.
func startBenchFrontline(b *testing.B, backendAddr string, pols []*frontlinev1.Policy, rowsBuffered *int64) (string, func()) {
	b.Helper()

	ps, err := proxy.New(proxy.Config{ //nolint:exhaustruct
		InstanceID:         "bench-instance",
		Platform:           "bench",
		Region:             "bench",
		ApexDomain:         "bench.local",
		Clock:              clock.New(),
		MaxHops:            3,
		UpstreamTransports: proxy.NewTransportRegistry(),
	})
	if err != nil {
		b.Fatal(err)
	}

	eng, err := policies.New(policies.Config{
		KeyService:       benchKeyService{},
		RateLimiter:      benchRatelimiter{},
		Clock:            clock.New(),
		KeyVerifications: batch.NewNoop[schema.KeyVerification](),
	})
	if err != nil {
		b.Fatal(err)
	}

	decision := localDecision(backendAddr)
	decision.Policies = pols

	h := &handler.Handler{
		RouterService: &stubRouter{decision: decision},
		ProxyService:  ps,
		Engine:        eng,
		Clock:         clock.New(),
	}

	buf := batch.New(batch.Config[schema.FrontlineRequest]{
		Name:          "bench",
		Drop:          true,
		BatchSize:     1000,
		BufferSize:    100000,
		FlushInterval: time.Hour,
		Consumers:     1,
		Flush: func(_ context.Context, rows []schema.FrontlineRequest) {
			atomic.AddInt64(rowsBuffered, int64(len(rows)))
		},
	})

	zenSrv, err := zen.New(zen.Config{ //nolint:exhaustruct
		ReadTimeout:        -1,
		WriteTimeout:       -1,
		MaxRequestBodySize: 0,
	})
	if err != nil {
		b.Fatal(err)
	}

	mws := []zen.Middleware{
		zen.WithPanicRecovery(),
		middleware.WithReservedHeaderStrip(),
		middleware.WithClickHouseLogging(buf, clock.New(), "fl_bench", "bench", "bench"),
		middleware.WithObservability(errorpage.NewRenderer()),
	}
	zenSrv.RegisterRoute(mws, h)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = zenSrv.Serve(ctx, ln) }()

	addr := ln.Addr().String()
	deadline := time.Now().Add(5 * time.Second)
	for {
		conn, dialErr := net.Dial("tcp", addr)
		if dialErr == nil {
			_ = conn.Close()
			break
		}
		if time.Now().After(deadline) {
			b.Fatalf("frontline did not start listening: %v", dialErr)
		}
		time.Sleep(5 * time.Millisecond)
	}

	return addr, func() {
		cancel()
		shutdownCtx, sc := context.WithTimeout(context.Background(), 2*time.Second)
		defer sc()
		_ = zenSrv.Shutdown(shutdownCtx)
		buf.Close()
	}
}

func runProxyBench(b *testing.B, pols []*frontlinev1.Policy, wantRows bool) {
	b.Helper()

	responseBody := bytes.Repeat([]byte("r"), 1024)
	backendAddr, stopBackend := startBenchBackend(b, responseBody)
	defer stopBackend()

	var rowsBuffered int64
	frontlineAddr, stop := startBenchFrontline(b, backendAddr, pols, &rowsBuffered)

	client := &http.Client{Transport: &http.Transport{MaxIdleConnsPerHost: 4}} //nolint:exhaustruct
	requestBody := bytes.Repeat([]byte("q"), 1024)
	url := "http://" + frontlineAddr + "/bench"

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(requestBody))
		if err != nil {
			b.Fatal(err)
		}
		req.Host = "bench.example.com"
		resp, err := client.Do(req)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := io.Copy(io.Discard, resp.Body); err != nil {
			b.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b.Fatalf("unexpected status %d", resp.StatusCode)
		}
	}
	b.StopTimer()

	// Close flushes remaining rows so the count is complete, then verify the
	// benchmark exercised the intended logging path.
	stop()
	got := atomic.LoadInt64(&rowsBuffered)
	if wantRows && got == 0 {
		b.Fatal("logging-enabled benchmark buffered no ClickHouse rows; the logging path was not exercised")
	}
	if !wantRows && got != 0 {
		b.Fatalf("logging-disabled benchmark buffered %d ClickHouse rows; logging should be off", got)
	}
}

func startBenchBackend(b *testing.B, body []byte) (string, func()) {
	b.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatal(err)
	}
	srv := &http.Server{ //nolint:exhaustruct
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.Copy(io.Discard, r.Body)
			w.Header().Set("Content-Length", fmt.Sprint(len(body)))
			_, _ = w.Write(body)
		}),
	}
	go func() { _ = srv.Serve(ln) }()
	return ln.Addr().String(), func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}
}

// BenchmarkProxy_LoggingDisabled measures the local-instance path with no
// logging policy: no body capture, no ClickHouse row.
func BenchmarkProxy_LoggingDisabled(b *testing.B) {
	runProxyBench(b, nil, false)
}

// BenchmarkProxy_LoggingEnabled measures the local-instance path with an
// enabled catch-all logging policy: bodies are captured and a ClickHouse row
// is buffered per request.
func BenchmarkProxy_LoggingEnabled(b *testing.B) {
	runProxyBench(b, []*frontlinev1.Policy{loggingPolicy()}, true)
}
