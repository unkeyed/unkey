package undns

import (
	"context"
	"io"
	"net"
	"sync"
	"time"

	dnswire "codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/deploy/appbinding"
	"github.com/unkeyed/unkey/pkg/logger"
)

const privateZone = appbinding.Zone + "."

type handler struct {
	catalog  *catalog
	config   Config
	slots    chan struct{}
	forwards chan struct{}
	queries  *prometheus.CounterVec
	shed     *prometheus.CounterVec
	duration prometheus.Histogram

	workspaceMu       sync.Mutex
	workspaceForwards map[string]int
}

func newHandler(c *catalog, config Config, registry *prometheus.Registry) *handler {
	queries := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "unkey_dns_queries_total",
		Help: "DNS responses by result and transport.",
	}, []string{"rcode", "transport"})
	duration := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "unkey_dns_query_duration_seconds",
		Help:    "DNS query latency including forwarding.",
		Buckets: prometheus.DefBuckets,
	})
	shed := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "unkey_dns_shed_total",
		Help: "Queries answered with SERVFAIL because a concurrency limit was full.",
	}, []string{"limit"})
	registry.MustRegister(queries, shed, duration)

	return &handler{
		catalog:           c,
		config:            config,
		slots:             make(chan struct{}, config.QueriesInFlight),
		forwards:          make(chan struct{}, config.ForwardsInFlight),
		queries:           queries,
		shed:              shed,
		duration:          duration,
		workspaceMu:       sync.Mutex{},
		workspaceForwards: make(map[string]int),
	}
}

func (h *handler) ServeDNS(ctx context.Context, w dnswire.ResponseWriter, request *dnswire.Msg) {
	start := time.Now()
	response := new(dnswire.Msg)
	dnsutil.SetReply(response, request)
	response.RecursionAvailable = true

	transport := "udp"
	if _, ok := w.RemoteAddr().(*net.TCPAddr); ok {
		transport = "tcp"
	}

	defer func() {
		size := 65535
		if transport == "udp" {
			size = dnswire.MinMsgSize
			if request.UDPSize > 0 {
				size = min(max(int(request.UDPSize), dnswire.MinMsgSize), 1232)
			}
		}

		if err := packTruncated(response, size); err != nil {
			logger.Warn("pack DNS response", "error", err)
			response = new(dnswire.Msg)
			dnsutil.SetReply(response, request)
			response.RecursionAvailable = true
			response.Rcode = dnswire.RcodeServerFailure
			if err := response.Pack(); err != nil {
				logger.Warn("pack DNS failure response", "error", err)
				return
			}
		}
		if _, err := io.Copy(w, response); err != nil {
			logger.Warn("write DNS response", "error", err)
		}

		h.queries.WithLabelValues(dnsutil.RcodeToString(response.Rcode), transport).Inc()
		h.duration.Observe(time.Since(start).Seconds())
	}()

	if err := request.Unpack(); err != nil {
		response.Rcode = dnswire.RcodeFormatError
		return
	}

	select {
	case h.slots <- struct{}{}:
	default:
		h.shed.WithLabelValues("queries").Inc()
		response.Rcode = dnswire.RcodeServerFailure
		return
	}

	local, workspace := h.answer(request, w.RemoteAddr())
	<-h.slots
	if local != nil {
		response = local
		return
	}

	response = h.forward(ctx, request, transport, workspace)
}
