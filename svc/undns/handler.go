package undns

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	dnswire "codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/deploy/appbinding"
	"github.com/unkeyed/unkey/pkg/logger"
)

const (
	privateZone = appbinding.Zone + "."

	pathPrivate = "private"
	pathPublic  = "public"
	pathInvalid = "invalid"

	failureLogInterval = time.Minute
)

var errQueriesFull = errors.New("queries concurrency limit is full")

type handler struct {
	catalog  *catalog
	config   Config
	slots    chan struct{}
	forwards chan struct{}
	queries  *prometheus.CounterVec
	shed     *prometheus.CounterVec
	duration *prometheus.HistogramVec
	failures *sampler

	workspaceMu       sync.Mutex
	workspaceForwards map[string]int
}

func newHandler(c *catalog, config Config, registry *prometheus.Registry) *handler {
	queries := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "unkey_dns_queries_total",
		Help: "DNS responses by zone path, the reason for the response code, response code, and transport.",
	}, []string{"path", "reason", "rcode", "transport"})
	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "unkey_dns_query_duration_seconds",
		Help:    "DNS query latency by zone path, including upstream forwarding for public names.",
		Buckets: []float64{0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
	}, []string{"path"})
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
		failures:          newSampler(failureLogInterval),
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

	path, result := pathInvalid, outcome{reason: reasonMalformed, err: nil, caller: emptyCaller()}
	defer func() {
		h.observe(path, result, response, transport, w.RemoteAddr(), time.Since(start))
	}()

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
			result = outcome{reason: reasonPackError, err: err, caller: result.caller}
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
	}()

	if err := request.Unpack(); err != nil {
		response.Rcode = dnswire.RcodeFormatError
		return
	}
	path = queryPath(request)

	select {
	case h.slots <- struct{}{}:
	default:
		h.shed.WithLabelValues("queries").Inc()
		response.Rcode = dnswire.RcodeServerFailure
		result = outcome{reason: reasonShed, err: errQueriesFull, caller: emptyCaller()}
		return
	}

	local, answered := h.answer(request, w.RemoteAddr())
	<-h.slots
	result = answered
	if local != nil {
		response = local
		return
	}

	response, result = h.forward(ctx, request, transport, answered.caller)
}

func queryPath(request *dnswire.Msg) string {
	if len(request.Question) != 1 || request.Response {
		return pathInvalid
	}
	if dnsutil.IsBelow(privateZone, strings.ToLower(request.Question[0].Header().Name)) {
		return pathPrivate
	}
	return pathPublic
}

func (h *handler) observe(path string, result outcome, response *dnswire.Msg, transport string, remote net.Addr, elapsed time.Duration) {
	rcode := dnsutil.RcodeToString(response.Rcode)
	h.queries.WithLabelValues(path, string(result.reason), rcode, transport).Inc()
	h.duration.WithLabelValues(path).Observe(elapsed.Seconds())

	failed := response.Rcode == dnswire.RcodeServerFailure || response.Rcode == dnswire.RcodeRefused
	if !failed || result.reason == reasonUpstream || !h.failures.allow(result.reason) {
		return
	}

	attrs := []any{
		"path", path, "reason", string(result.reason), "rcode", rcode, "transport", transport,
		"source", remote.String(), "workspace_id", result.caller.workspace,
		"caller_deployment_id", result.caller.deployment, "error", result.err,
	}
	if path == pathPrivate && len(response.Question) == 1 {
		attrs = append(attrs, "name", response.Question[0].Header().Name)
	}
	logger.Warn("DNS query failed", attrs...)
}
