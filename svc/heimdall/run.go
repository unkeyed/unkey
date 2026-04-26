package heimdall

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/runner"
	"github.com/unkeyed/unkey/svc/heimdall/internal/collector"
	"github.com/unkeyed/unkey/svc/heimdall/internal/network"
	"github.com/unkeyed/unkey/svc/heimdall/internal/metrics"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

func Run(ctx context.Context, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

	// Refuse to start if this node isn't billable (wrong cgroup version or
	// no kubelet-managed cgroup tree). Silent undercharge is worse than
	// not starting. Both systemd (prod EKS) and cgroupfs (dev minikube)
	// drivers are accepted; we detect and adapt.
	cgroupDriver, err := collector.Preflight("/sys/fs/cgroup")
	if err != nil {
		return fmt.Errorf("preflight check failed: %w", err)
	}

	logger.Info("starting heimdall",
		"region", cfg.Region,
		"platform", cfg.Platform,
		"collection_interval", cfg.CollectionInterval.String(),
		"cgroup_driver", cgroupDriver.String(),
	)

	ch, err := clickhouse.New(clickhouse.Config{URL: cfg.ClickHouseURL})
	if err != nil {
		return fmt.Errorf("unable to create clickhouse client: %w", err)
	}

	// Drop: false, so we backpressure the checkpoint loop if ClickHouse is
	// unreachable. Losing checkpoints would be a silent undercharge; blocking
	// the loop gives us visibility that something is wrong.
	//
	// Consumers: 2 so burst-ingest after an outage (50k-row backlog drains in
	// parallel) reaches ClickHouse faster, which in turn lets the dashboard
	// MVs catch up sooner — they fire per INSERT and lag visibly when one
	// consumer serializes a big backlog. async_insert_deduplicate=1 makes
	// concurrent inserts safe; dedup is by content hash, no ordering
	// assumption. Matches the codebase idiom (svc/api uses 2 for its CH
	// buffers).
	checkpointBuffer := clickhouse.NewBuffer[schema.InstanceCheckpoint](ch, "default.instance_checkpoints_v1", clickhouse.BufferConfig{
		Name:          "instance_checkpoints",
		Drop:          false,
		BatchSize:     10_000,
		BufferSize:    50_000,
		FlushInterval: 5 * time.Second,
		Consumers:     2,
		OnFlushError:  nil,
	})

	k8sCfg, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("getting in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(k8sCfg)
	if err != nil {
		return fmt.Errorf("creating kubernetes client: %w", err)
	}

	factory := informers.NewSharedInformerFactory(clientset, 0)
	podInformer := factory.Core().V1().Pods().Informer()
	podLister := factory.Core().V1().Pods().Lister()

	factory.Start(ctx.Done())

	// Refuse to start the collector with an unsynced cache. Without the
	// check, a sync failure (ctx cancel before initial List completes, API
	// server unreachable) would leave the lister returning empty results,
	// every pod on the node would be silently unbilled.
	synced := factory.WaitForCacheSync(ctx.Done())
	for resource, ok := range synced {
		if !ok {
			return fmt.Errorf("informer cache sync failed for %v", resource)
		}
	}

	metrics.InformerCacheSynced.Set(1)

	// Network byte counters via tc eBPF on each pod's host-side veth.
	// On Linux this loads the program once and dials containerd at
	// cfg.CRISocket for sandbox netns resolution (sandbox container's OCI
	// spec carries the /var/run/netns/cni-<uuid> path uniformly for runc
	// and gVisor). On non-Linux (macOS dev/test) NewReader returns a
	// no-op stub. Failure to load or dial is fatal: we'd silently
	// undercharge every pod for network usage.
	netReader, err := network.NewReader(cfg.CRISocket)
	if err != nil {
		return fmt.Errorf("loading network ebpf program: %w", err)
	}

	kc := collector.New(collector.Config{
		CH:           checkpointBuffer,
		PodLister:    podLister,
		CgroupRoot:   "/sys/fs/cgroup",
		CgroupDriver: cgroupDriver,
		KubeletRoot:  cfg.KubeletRoot,
		NodeName:     cfg.NodeName,
		Region:       cfg.Region,
		Platform:     cfg.Platform,
		CRISocket:    cfg.CRISocket,
		Clock:        clock.NewMonotonic(),
		Network:      netReader,
	})

	// Pod informer UpdateFunc as a secondary lifecycle source. CRI is the
	// primary (ms precision); the informer catches cases where containerd
	// drops an event (containerd#3177) or CRI is unavailable. Duplicates
	// across both paths are absorbed by max-min billing math.
	if _, err := podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: nil,
		UpdateFunc: func(oldObj, newObj any) {
			oldPod, ok1 := oldObj.(*corev1.Pod)
			newPod, ok2 := newObj.(*corev1.Pod)
			if !ok1 || !ok2 {
				return
			}
			kc.HandlePodUpdate(oldPod, newPod)
		},
		DeleteFunc: nil,
	}); err != nil {
		return fmt.Errorf("registering pod event handler: %w", err)
	}

	r := runner.New()

	// Single HTTP server on `metrics.port` serving both Prometheus metrics
	// and the kubelet probe endpoints. Same port because they share the
	// same "is the collector reachable?" semantics — if /metrics is up,
	// /health/* is up — and doubling the hostNetwork port allocation
	// isn't worth the separation. hostNetwork is true in k8s, so the
	// port has to coexist with node-exporter (9100) and anything else
	// the node runs. The chart defaults to 9402.
	//
	// Paths:
	//   GET /metrics        Prometheus scrape endpoint (promhttp.Handler)
	//   GET /health/live    kubelet liveness probe
	//   GET /health/ready   kubelet readiness probe (informer cache synced)
	//   GET /health/startup kubelet startup probe
	//
	// Gated on a non-zero port via [observability.metrics] prometheus_port;
	// omit the section in dev to run without a scraper or probes.
	if cfg.Observability.Metrics != nil && cfg.Observability.Metrics.PrometheusPort > 0 {
		// Preflight (cgroup driver), informer cache sync, and BPF attach
		// all run synchronously above — by the time r.Wait() flips
		// health.started to true, those are all done. No additional
		// readiness check needed: /ready returns 200 if the runner
		// started and isn't shutting down, which already captures the
		// "collector is live and initialized" condition.
		mux := http.NewServeMux()
		mux.Handle("GET /metrics", promhttp.Handler())
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

	// Buffer + CH lifecycle are tied to the collector goroutine so shutdown
	// order is: ctx cancel → kc.Run drains in-flight handlers → buffer.Close
	// flushes the batch → ch.Close tears down the connection. Doing these in
	// r.Defer would run them BEFORE the runner's wg.Wait, racing with any
	// late handler writes.
	r.Go(func(ctx context.Context) error {
		defer func() {
			if err := ch.Close(); err != nil {
				logger.Warn("clickhouse close failed", "error", err.Error())
			}
		}()

		defer checkpointBuffer.Close()
		defer func() {
			if err := netReader.Close(); err != nil {
				logger.Warn("network reader close failed", "error", err.Error())
			}
		}()

		return kc.Run(ctx, cfg.CollectionInterval)
	})

	return r.Wait(ctx)
}
