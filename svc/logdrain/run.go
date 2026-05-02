package logdrain

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
	vaultrpc "github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/pkg/buildinfo"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
	"github.com/unkeyed/unkey/pkg/rpc/interceptor"
	"github.com/unkeyed/unkey/pkg/runner"
	"github.com/unkeyed/unkey/svc/logdrain/internal/coordinator"
	"github.com/unkeyed/unkey/svc/logdrain/internal/creds"
)

// Run boots the logdrain service. It dials ClickHouse, MySQL, and Vault,
// constructs the coordinator, and blocks on ctx. Sink fan-out for live
// CH batches lands in the next stack; v1 of the loop logs the groups it
// would process so the wiring is exercisable end-to-end without sending
// real records to the providers.
func Run(ctx context.Context, cfg Config) error {
	// The pod's ordinal comes from $HOSTNAME (StatefulSet pattern
	// `<name>-<ordinal>`) so the helm chart doesn't have to inject a
	// per-pod value. Local dev and tests run as a single replica with
	// ordinal=0.
	ordinal, _ := podOrdinalFromEnv()

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("bad config: %w", err)
	}
	if ordinal >= cfg.Replicas {
		return fmt.Errorf(
			"pod ordinal (%d) must be < replicas (%d) — set replicas in the ConfigMap before scaling pods up",
			ordinal, cfg.Replicas,
		)
	}

	shardStart, shardEnd := coordinator.ShardRange(ordinal, cfg.Replicas)

	logger.Info("starting logdrain",
		"ordinal", ordinal,
		"replicas", cfg.Replicas,
		"shard_start", shardStart,
		"shard_end", shardEnd,
		"total_shards", coordinator.TotalShards,
		"poll_interval", cfg.PollInterval.String(),
	)

	// Install a private Prometheus registry so the lazy instruments
	// declared in svc/logdrain/internal/metrics flush their buffered
	// writes against this service's metrics handler instead of the
	// process-default registry. Same shape as svc/ctrl/api/run.go.
	reg := promclient.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	//nolint:exhaustruct // ProcessCollectorOpts zero values are fine.
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	lazy.SetRegistry(reg)
	buildinfo.RegisterBuildInfoMetrics("logdrain")

	ch, err := clickhouse.New(clickhouse.Config{URL: cfg.ClickHouseURL})
	if err != nil {
		return fmt.Errorf("unable to create clickhouse client: %w", err)
	}

	database, err := db.New(db.Config{
		PrimaryDSN:  cfg.Database.Primary,
		ReadOnlyDSN: cfg.Database.ReadonlyReplica,
	})
	if err != nil {
		return fmt.Errorf("unable to create db: %w", err)
	}

	vaultClient := vaultrpc.NewConnectVaultServiceClient(vaultv1connect.NewVaultServiceClient(
		&http.Client{Timeout: 10 * time.Second},
		cfg.Vault.URL,
		connect.WithInterceptors(interceptor.NewHeaderInjector(map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", cfg.Vault.Token),
		})),
	))

	// Honor the operator-configured TTL. Stale defaults to 2× Fresh so
	// SWR (serve-while-revalidate) gives one free refresh window per TTL
	// without an extra knob — matches the pattern in pkg/cache.
	credsCache, err := creds.NewCache(vaultClient, creds.Config{
		MaxSize: 0,
		Fresh:   cfg.CredentialCacheTTL,
		Stale:   2 * cfg.CredentialCacheTTL,
		Clock:   nil,
	})
	if err != nil {
		return fmt.Errorf("unable to create creds cache: %w", err)
	}
	// A drain typically hits the same provider host (api.axiom.co, ...)
	// many times per tick. Go's
	// default Transport caps idle conns per host at 2, which thrashes
	// the connection pool and forces a TLS handshake on most requests.
	// Bumping MaxIdleConnsPerHost plus an explicit IdleConnTimeout
	// keeps the pool warm for the typical 10s poll interval and pays
	// off most for fan-outs of 5+ drains per group.
	httpClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: newSinkTransport(),
	}
	factory := coordinator.NewFactory(credsCache, httpClient)

	coord, err := coordinator.New(coordinator.Config{
		PollInterval:          cfg.PollInterval,
		BatchWindow:           cfg.BatchWindow,
		MaxBatchRecords:       cfg.MaxBatchRecords,
		PauseAfterFailures:    cfg.PauseAfterFailures,
		MaxGroupsPerShard:     cfg.MaxGroupsPerShard,
		MaxDrainsPerWorkspace: cfg.MaxDrainsPerWorkspace,
		Ordinal:               ordinal,
		ShardStart:            shardStart,
		ShardEnd:              shardEnd,
	}, database, ch, factory)
	if err != nil {
		return fmt.Errorf("unable to create coordinator: %w", err)
	}

	r := runner.New()

	// Single HTTP server on metrics.port serving Prometheus metrics and
	// kubelet probes — same shape heimdall uses. Gated on a non-zero port
	// so dev runs without a scraper.
	if cfg.Observability.Metrics != nil && cfg.Observability.Metrics.PrometheusPort > 0 {
		mux := http.NewServeMux()
		//nolint:exhaustruct // HandlerOpts zero values are fine; we want the defaults.
		mux.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
		r.RegisterHealth(mux, "/health")

		addr := fmt.Sprintf(":%d", cfg.Observability.Metrics.PrometheusPort)
		listener, listenErr := net.Listen("tcp", addr)
		if listenErr != nil {
			return fmt.Errorf("unable to listen on %s: %w", addr, listenErr)
		}

		server := &http.Server{
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		}

		logger.Info("starting metrics+health server", "addr", addr)

		r.DeferCtx(func(shutdownCtx context.Context) error {
			return server.Shutdown(shutdownCtx)
		})
		r.Go(func(_ context.Context) error {
			serveErr := server.Serve(listener)
			if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
				return fmt.Errorf("metrics+health server failed: %w", serveErr)
			}
			return nil
		})
	}

	r.Go(func(ctx context.Context) error {
		defer func() {
			if cerr := ch.Close(); cerr != nil {
				logger.Warn("clickhouse close failed", "error", cerr.Error())
			}
			if cerr := database.Close(); cerr != nil {
				logger.Warn("database close failed", "error", cerr.Error())
			}
		}()
		return coord.Run(ctx)
	})

	return r.Wait(ctx)
}

// newSinkTransport returns an http.RoundTripper tuned for the logdrain
// fan-out: a small set of provider hostnames hit many times per second.
// The numbers are deliberate copies of http.DefaultTransport with the
// per-host idle pool sized for the worst case (max group concurrency ×
// drains-per-group of the same provider) and a longer idle timeout so
// connections survive between ticks. Anything stricter than 90s burns
// TLS handshakes during quiet periods.
func newSinkTransport() *http.Transport {
	//nolint:exhaustruct // matches svc/frontline/internal/proxy/transport.go
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          256,
		MaxIdleConnsPerHost:   64,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// podOrdinalFromEnv recovers the StatefulSet ordinal from $HOSTNAME, the
// pattern Kubernetes guarantees for `<statefulset-name>-<ordinal>`. We
// walk the suffix instead of regex-matching because the pod name shape
// is rigid and a substring match keeps the failure mode obvious: any
// hostname whose suffix isn't `-N` returns ok=false and the caller
// keeps the configured shard_index untouched.
func podOrdinalFromEnv() (int, bool) {
	host := strings.TrimSpace(os.Getenv("HOSTNAME"))
	if host == "" {
		return 0, false
	}
	idx := strings.LastIndex(host, "-")
	if idx < 0 || idx == len(host)-1 {
		return 0, false
	}
	ordinal, err := strconv.Atoi(host[idx+1:])
	if err != nil || ordinal < 0 {
		return 0, false
	}
	return ordinal, true
}
