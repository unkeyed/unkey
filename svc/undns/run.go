package undns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	dnswire "codeberg.org/miekg/dns"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/unkeyed/unkey/pkg/runner"
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
	c, err := newCatalog(client, cfg.WatchTimeout)
	if err != nil {
		return err
	}
	return serve(ctx, cfg, c)
}

func serve(ctx context.Context, cfg Config, c *catalog) error {
	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "unkey_dns_discovery_ready",
		Help: "Whether private discovery is synchronized, fresh, and activated.",
	}, func() float64 {
		if c.ready() {
			return 1
		}
		return 0
	}))
	h := newHandler(c, cfg, registry)
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
	r := runner.New()
	r.Go(c.run)
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	r.RegisterHealth(mux)
	httpServer := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	r.DeferCtx(httpServer.Shutdown)
	r.Go(func(context.Context) error {
		if err := httpServer.Serve(health); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})
	tcpServer := new(dnswire.Server)
	tcpServer.Listener = tcp
	udpServer := new(dnswire.Server)
	udpServer.PacketConn = udp
	for transport, server := range map[string]*dnswire.Server{"tcp": tcpServer, "udp": udpServer} {
		server.Handler = h
		server.UDPSize = 1232
		server.ReadTimeout = 5 * time.Second
		started := make(chan struct{})
		finished := make(chan struct{})
		server.NotifyStartedFunc = func(context.Context) { close(started) }
		r.AddReadinessCheck("dns_"+transport, func(context.Context) error {
			select {
			case <-finished:
				return fmt.Errorf("DNS listener stopped")
			default:
			}
			select {
			case <-started:
				return nil
			default:
				return fmt.Errorf("DNS listener has not started")
			}
		})
		r.Go(func(context.Context) error {
			defer close(finished)
			return server.ListenAndServe()
		})
		r.DeferCtx(func(shutdownCtx context.Context) error {
			select {
			case <-started:
				server.Shutdown(shutdownCtx)
				return nil
			case <-finished:
				return nil
			case <-shutdownCtx.Done():
				return shutdownCtx.Err()
			}
		})
	}
	return r.Wait(ctx)
}
