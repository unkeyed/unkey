package undns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/prometheus"
	"github.com/unkeyed/unkey/pkg/runner"
	"github.com/unkeyed/unkey/svc/undns/internal/discovery"
	"github.com/unkeyed/unkey/svc/undns/internal/resolver"
	"github.com/unkeyed/unkey/svc/undns/internal/server"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Run serves regional DNS with in-cluster, read-only Kubernetes discovery.
func Run(ctx context.Context, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	kubeConfig, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("load Kubernetes configuration: %w", err)
	}
	client, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return fmt.Errorf("create Kubernetes client: %w", err)
	}
	catalog, err := discovery.New(client, cfg.WatchTimeout, clock.New())
	if err != nil {
		return err
	}
	return serve(ctx, cfg, catalog)
}

func serve(ctx context.Context, cfg Config, catalog *discovery.Catalog) error {
	tcp, err := net.Listen("tcp", cfg.ListenAddress)
	if err != nil {
		return fmt.Errorf("listen for TCP DNS: %w", err)
	}
	udp, err := net.ListenPacket("udp", cfg.ListenAddress)
	if err != nil {
		return errors.Join(fmt.Errorf("listen for UDP DNS: %w", err), tcp.Close())
	}
	health, err := net.Listen("tcp", cfg.HealthAddress)
	if err != nil {
		return errors.Join(fmt.Errorf("listen for DNS health: %w", err), tcp.Close(), udp.Close())
	}
	return serveListeners(ctx, cfg, catalog, tcp, udp, health)
}

func serveListeners(ctx context.Context, cfg Config, catalog *discovery.Catalog, tcp net.Listener, udp net.PacketConn, health net.Listener) error {
	registry := prometheus.NewServiceRegistry()
	dns, err := resolver.New(catalog, resolver.Config{
		Upstream:             cfg.Upstream,
		TTLSeconds:           cfg.TTLSeconds,
		ForwardTimeout:       cfg.ForwardTimeout,
		QueriesInFlight:      cfg.QueriesInFlight,
		ForwardsInFlight:     cfg.ForwardsInFlight,
		ForwardsPerWorkspace: cfg.ForwardsPerWorkspace,
		ForwardsUnidentified: cfg.ForwardsUnidentified,
		ForwardCacheEntries:  cfg.ForwardCacheEntries,
		Clock:                clock.New(),
	})
	if err != nil {
		return errors.Join(err, tcp.Close(), udp.Close(), health.Close())
	}

	r := runner.New()
	r.Defer(func() error {
		dns.Close()
		return nil
	})
	r.Go(catalog.Run)

	mux := http.NewServeMux()
	mux.Handle("GET /metrics", prometheus.Handler(registry))
	r.RegisterHealth(mux)
	httpServer := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	r.DeferCtx(httpServer.Shutdown)
	r.Go(func(context.Context) error {
		if err := httpServer.Serve(health); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	tcpDNS := server.NewTCP(tcp, dns)
	r.Go(tcpDNS.Serve)
	r.DeferCtx(tcpDNS.Shutdown)
	r.AddReadinessCheck("dns_tcp", func(ctx context.Context) error {
		return server.Probe(ctx, "tcp", tcp.Addr().String())
	})

	udpDNS := server.NewUDP(udp, dns)
	r.Go(udpDNS.Serve)
	r.DeferCtx(udpDNS.Shutdown)
	r.AddReadinessCheck("dns_udp", func(ctx context.Context) error {
		return server.Probe(ctx, "udp", udp.LocalAddr().String())
	})

	return r.Wait(ctx)
}
