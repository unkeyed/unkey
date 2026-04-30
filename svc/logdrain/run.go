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

	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/unkeyed/unkey/pkg/buildinfo"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
	"github.com/unkeyed/unkey/pkg/runner"
)

// Run boots the logdrain service. v1 is a skeleton: it validates config,
// dials ClickHouse, exposes /metrics + /health, and blocks on ctx. The
// coordinator and worker loops land in a follow-up commit.
func Run(ctx context.Context, cfg Config) error {
	// When the pod is part of a StatefulSet, prefer the ordinal we can
	// derive from $HOSTNAME so the helm chart doesn't have to inject one
	// shard_index per replica. The cfg value still wins when no ordinal
	// is detected (single-replica Deployment, local dev, tests).
	if ordinal, ok := podOrdinalFromEnv(); ok {
		if ordinal != cfg.ShardIndex {
			logger.Info("overriding shard_index from pod ordinal",
				"config_shard_index", cfg.ShardIndex,
				"pod_ordinal", ordinal,
			)
		}
		cfg.ShardIndex = ordinal
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

	logger.Info("starting logdrain",
		"shard_index", cfg.ShardIndex,
		"shard_count", cfg.ShardCount,
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

	// Skeleton main loop: hold the ClickHouse handle, surface readiness,
	// and block on ctx until the coordinator lands. Closing CH on exit so
	// the binary is well-behaved on SIGTERM today.
	r.Go(func(ctx context.Context) error {
		defer func() {
			if err := ch.Close(); err != nil {
				logger.Warn("clickhouse close failed", "error", err.Error())
			}
		}()

		<-ctx.Done()
		return nil
	})

	return r.Wait(ctx)
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
