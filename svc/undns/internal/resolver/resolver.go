package resolver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"

	dnswire "codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/undns/internal/discovery"
	"github.com/unkeyed/unkey/svc/undns/pkg/metrics"
)

const (
	privateZone = privatenetwork.Zone + "."

	pathPrivate = "private"
	pathPublic  = "public"
	pathInvalid = "invalid"

	failureLogInterval = time.Minute
)

var errQueriesFull = errors.New("queries concurrency limit is full")

// Catalog is the discovery state a [Resolver] answers private names from.
// [*discovery.Catalog] implements it.
type Catalog interface {
	Ready() bool
	IdentityReady() bool
	Identify(ip netip.Addr) (discovery.Caller, error)
	Resolve(caller discovery.Caller, name string) ([]netip.Addr, bool, error)
}

// Config bounds a [Resolver]. QueriesInFlight and ForwardsInFlight must be
// positive, ForwardCacheEntries must be at least [MinForwardCacheEntries], and
// the forward quotas must not exceed ForwardsInFlight.
type Config struct {
	// Upstream is the address public names are forwarded to.
	Upstream string
	// TTLSeconds is the TTL of private answers and of the private zone SOA.
	TTLSeconds uint32
	// ForwardTimeout bounds each upstream exchange.
	ForwardTimeout time.Duration
	// QueriesInFlight bounds queries doing local work.
	QueriesInFlight int
	// ForwardsInFlight bounds upstream forwards across all callers.
	ForwardsInFlight int
	// ForwardsPerWorkspace bounds the forwards of one caller workspace.
	ForwardsPerWorkspace int
	// ForwardsUnidentified bounds the forwards shared by callers without a
	// fresh Pod identity.
	ForwardsUnidentified int
	// ForwardCacheEntries bounds the public response cache.
	ForwardCacheEntries int
	// Clock supplies time for cache expiry and failure log sampling.
	Clock clock.Clock
}

// Resolver answers DNS queries: names below the private zone from a [Catalog]
// scoped to the caller, and every other name from the upstream resolver. It
// implements dnswire.Handler and is safe for concurrent use. Call
// [Resolver.Close] after the servers using it have stopped.
type Resolver struct {
	catalog  Catalog
	config   Config
	slots    chan struct{}
	forwards chan struct{}
	failures *sampler
	client   *dnswire.Client
	cache    *forwardCache

	workspaceMu       sync.Mutex
	workspaceForwards map[string]int
}

// New returns a Resolver answering from catalog. It fails if cfg is invalid or
// the forward cache can't be created.
func New(catalog Catalog, cfg Config) (*Resolver, error) {
	err := assert.All(
		assert.NotNil(cfg.Clock, "clock is required"),
		assert.Greater(cfg.QueriesInFlight, 0, "queries_in_flight must be positive"),
		assert.Greater(cfg.ForwardsInFlight, 0, "forwards_in_flight must be positive"),
		assert.GreaterOrEqual(cfg.ForwardCacheEntries, MinForwardCacheEntries, fmt.Sprintf("forward_cache_entries must be at least %d", MinForwardCacheEntries)),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid resolver config: %w", err)
	}
	cache, err := newForwardCache(cfg.ForwardCacheEntries, cfg.Clock)
	if err != nil {
		return nil, err
	}

	dialer := new(net.Dialer)
	dialer.Timeout = cfg.ForwardTimeout
	transport := dnswire.NewTransport()
	transport.Dialer = dialer
	transport.ReadTimeout = cfg.ForwardTimeout
	transport.WriteTimeout = cfg.ForwardTimeout

	metrics.UpstreamHealthy.Set(0)
	return &Resolver{
		catalog:           catalog,
		config:            cfg,
		slots:             make(chan struct{}, cfg.QueriesInFlight),
		forwards:          make(chan struct{}, cfg.ForwardsInFlight),
		failures:          newSampler(failureLogInterval, cfg.Clock),
		client:            &dnswire.Client{Transport: transport, Transfer: nil},
		cache:             cache,
		workspaceMu:       sync.Mutex{},
		workspaceForwards: make(map[string]int),
	}, nil
}

// Close releases the forward cache's background workers.
func (r *Resolver) Close() {
	r.cache.close()
}

// ServeDNS answers request on w. A full concurrency limit answers SERVFAIL
// immediately. Every response is counted in unkey_dns_queries_total, and
// SERVFAIL and REFUSED responses are logged at most once per reason per
// minute.
func (r *Resolver) ServeDNS(ctx context.Context, w dnswire.ResponseWriter, request *dnswire.Msg) {
	start := time.Now()
	response := new(dnswire.Msg)
	dnsutil.SetReply(response, request)
	response.RecursionAvailable = true

	transport := "udp"
	if _, ok := w.RemoteAddr().(*net.TCPAddr); ok {
		transport = "tcp"
	}

	path, result := pathInvalid, outcome{reason: reasonMalformed, err: nil, caller: noCaller()}
	defer func() {
		r.observe(path, result, response, transport, w.RemoteAddr(), time.Since(start))
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
			logger.Warn("pack DNS response", "error_type", fmt.Sprintf("%T", err))
			result = outcome{reason: reasonPackError, err: err, caller: result.caller}
			response = new(dnswire.Msg)
			dnsutil.SetReply(response, request)
			response.RecursionAvailable = true
			response.Rcode = dnswire.RcodeServerFailure
			if err := response.Pack(); err != nil {
				logger.Warn("pack DNS failure response", "error_type", fmt.Sprintf("%T", err))
				return
			}
		}
		if _, err := io.Copy(w, response); err != nil {
			logger.Warn("write DNS response", "error_type", fmt.Sprintf("%T", err))
		}
	}()

	if err := request.Unpack(); err != nil {
		response.Rcode = dnswire.RcodeFormatError
		return
	}
	path = queryPath(request)

	select {
	case r.slots <- struct{}{}:
	default:
		metrics.ShedTotal.WithLabelValues("queries").Inc()
		response.Rcode = dnswire.RcodeServerFailure
		result = outcome{reason: reasonShed, err: errQueriesFull, caller: noCaller()}
		return
	}

	local, answered := r.answer(request, w.RemoteAddr())
	<-r.slots
	result = answered
	if local != nil {
		response = local
		return
	}

	response, result = r.forward(ctx, request, transport, answered.caller)
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

func (r *Resolver) observe(path string, result outcome, response *dnswire.Msg, transport string, remote net.Addr, elapsed time.Duration) {
	rcode := dnsutil.RcodeToString(response.Rcode)
	metrics.QueriesTotal.WithLabelValues(path, string(result.reason), rcode, transport).Inc()
	metrics.QueryDurationSeconds.WithLabelValues(path).Observe(elapsed.Seconds())

	failed := response.Rcode == dnswire.RcodeServerFailure || response.Rcode == dnswire.RcodeRefused
	if !failed || result.reason == reasonUpstream || !r.failures.allow(result.reason) {
		return
	}

	logger.Warn("DNS query failed",
		"path", path, "reason", string(result.reason), "rcode", rcode, "transport", transport,
		"source", remote.String(), "workspace_id", result.caller.Workspace,
		"caller_deployment_id", result.caller.Deployment, "error_type", fmt.Sprintf("%T", result.err),
	)
}
