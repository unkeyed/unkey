package undns

import (
	"context"
	"io"
	"math/rand/v2"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"

	dnswire "codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
	"codeberg.org/miekg/dns/rdata"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/logger"
	privatecontract "github.com/unkeyed/unkey/pkg/privatenetwork"
)

const privateZone = privatecontract.Zone + "."

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

func (h *handler) answer(request *dnswire.Msg, remote net.Addr) (*dnswire.Msg, string) {
	response := new(dnswire.Msg)
	dnsutil.SetReply(response, request)
	response.RecursionAvailable = true

	if len(request.Question) != 1 || request.Response {
		response.Rcode = dnswire.RcodeFormatError
		return response, ""
	}

	question := request.Question[0]
	if request.Opcode != dnswire.OpcodeQuery || question.Header().Class != dnswire.ClassINET {
		response.Rcode = dnswire.RcodeNotImplemented
		return response, ""
	}

	podsHealthy := h.catalog.pods.healthy()
	identity, identified := emptyCaller(), false
	if podsHealthy {
		if ip, err := netip.ParseAddrPort(remote.String()); err == nil {
			identity, identified = h.catalog.identify(ip.Addr())
		}
	}

	name := strings.ToLower(question.Header().Name)
	if !dnsutil.IsBelow(privateZone, name) {
		return nil, identity.workspace
	}

	if !podsHealthy {
		response.Rcode = dnswire.RcodeServerFailure
		return response, ""
	}

	if !identified {
		response.Rcode = dnswire.RcodeRefused
		return response, ""
	}

	if !h.catalog.ready() {
		response.Rcode = dnswire.RcodeServerFailure
		return response, ""
	}

	h.answerPrivate(response, question, identity)
	return response, ""
}

func (h *handler) answerPrivate(response *dnswire.Msg, question dnswire.RR, identity caller) {
	name := strings.ToLower(question.Header().Name)
	qtype := dnswire.RRToType(question)
	response.Authoritative = true
	app := strings.TrimSuffix(name, "."+privateZone)
	if name == privateZone {
		if qtype == dnswire.TypeSOA {
			response.Answer = []dnswire.RR{h.soa()}
		} else {
			response.Ns = []dnswire.RR{h.soa()}
		}
		return
	}

	if strings.Contains(app, ".") {
		response.Rcode = dnswire.RcodeNameError
		response.Ns = []dnswire.RR{h.soa()}
		return
	}

	addresses, exists, err := h.catalog.resolve(identity, app)
	if err != nil {
		response.Rcode = dnswire.RcodeServerFailure
		return
	}

	if !exists {
		response.Rcode = dnswire.RcodeNameError
		response.Ns = []dnswire.RR{h.soa()}
		return
	}

	if qtype != dnswire.TypeA {
		if qtype == dnswire.TypeAAAA {
			response.Ns = []dnswire.RR{h.soa()}
		} else {
			response.Rcode = dnswire.RcodeNotImplemented
		}
		return
	}

	rand.Shuffle(len(addresses), func(i, j int) { addresses[i], addresses[j] = addresses[j], addresses[i] })
	for _, address := range addresses {
		response.Answer = append(response.Answer, &dnswire.A{
			Hdr: dnswire.Header{Name: question.Header().Name, Class: dnswire.ClassINET, TTL: h.config.TTLSeconds},
			A:   rdata.A{Addr: address},
		})
	}
}

func (h *handler) soa() *dnswire.SOA {
	return &dnswire.SOA{
		Hdr: dnswire.Header{Name: privateZone, Class: dnswire.ClassINET, TTL: h.config.TTLSeconds},
		SOA: rdata.SOA{
			Ns: "ns." + privateZone, Mbox: "hostmaster." + privateZone, Serial: 1,
			Refresh: 30, Retry: 5, Expire: 60, Minttl: h.config.TTLSeconds,
		},
	}
}

func (h *handler) acquireForward(workspace string) bool {
	select {
	case h.forwards <- struct{}{}:
	default:
		h.shed.WithLabelValues("forwards").Inc()
		return false
	}

	h.workspaceMu.Lock()
	defer h.workspaceMu.Unlock()
	limit, metric := h.config.ForwardsPerWorkspace, "forwards_per_workspace"
	if workspace == "" {
		limit, metric = h.config.ForwardsUnidentified, "forwards_unidentified"
	}
	if h.workspaceForwards[workspace] >= limit {
		<-h.forwards
		h.shed.WithLabelValues(metric).Inc()
		return false
	}
	h.workspaceForwards[workspace]++
	return true
}

func (h *handler) releaseForward(workspace string) {
	h.workspaceMu.Lock()
	if h.workspaceForwards[workspace] <= 1 {
		delete(h.workspaceForwards, workspace)
	} else {
		h.workspaceForwards[workspace]--
	}
	h.workspaceMu.Unlock()
	<-h.forwards
}

func (h *handler) forward(ctx context.Context, request *dnswire.Msg, transport, workspace string) *dnswire.Msg {
	if !h.acquireForward(workspace) {
		response := new(dnswire.Msg)
		dnsutil.SetReply(response, request)
		response.Rcode = dnswire.RcodeServerFailure
		response.RecursionAvailable = true
		return response
	}
	defer h.releaseForward(workspace)

	query := dnswire.NewMsg(request.Question[0].Header().Name, dnswire.RRToType(request.Question[0]))
	query.RecursionDesired = request.RecursionDesired
	query.UDPSize = 1232

	dialer := new(net.Dialer)
	dialer.Timeout = h.config.ForwardTimeout
	transportConfig := dnswire.NewTransport()
	transportConfig.Dialer = dialer
	transportConfig.ReadTimeout = h.config.ForwardTimeout
	transportConfig.WriteTimeout = h.config.ForwardTimeout
	client := dnswire.NewClient()
	client.Transport = transportConfig
	ctx, cancel := context.WithTimeout(ctx, h.config.ForwardTimeout)
	defer cancel()

	response, _, err := client.Exchange(ctx, query, transport, h.config.Upstream)
	if err == nil && response.Truncated && transport == "udp" {
		response, _, err = client.Exchange(ctx, query, "tcp", h.config.Upstream)
	}

	if err != nil || response == nil || !response.Response || response.Opcode != dnswire.OpcodeQuery ||
		len(response.Question) != 1 || !sameQuestion(response.Question[0], query.Question[0]) {
		response = new(dnswire.Msg)
		dnsutil.SetReply(response, request)
		response.Rcode = dnswire.RcodeServerFailure
		response.RecursionAvailable = true
		return response
	}

	response.ID = request.ID
	return response
}

func sameQuestion(a, b dnswire.RR) bool {
	return a.Header().Name == b.Header().Name && a.Header().Class == b.Header().Class && dnswire.RRToType(a) == dnswire.RRToType(b)
}

func packTruncated(response *dnswire.Msg, size int) error {
	original := response.Answer
	if err := response.Pack(); err != nil {
		return err
	}
	if len(response.Data) <= size {
		return nil
	}

	low, high := 0, len(original)
	for low < high {
		middle := (low + high + 1) / 2
		response.Answer = original[:middle]
		response.Truncated = middle < len(original)
		if err := response.Pack(); err != nil {
			return err
		}
		if len(response.Data) <= size {
			low = middle
		} else {
			high = middle - 1
		}
	}
	response.Answer = original[:low]
	response.Truncated = true
	if err := response.Pack(); err != nil {
		return err
	}
	if len(response.Data) > size {
		response.Ns = nil
		response.Extra = nil
		return response.Pack()
	}
	return nil
}
