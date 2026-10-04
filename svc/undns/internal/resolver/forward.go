package resolver

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"

	dnswire "codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/undns/internal/discovery"
	"github.com/unkeyed/unkey/svc/undns/pkg/metrics"
)

type forwardResult struct {
	response *dnswire.Msg
	reason   reason
	err      error
}

func (r *Resolver) acquireForward(workspace string) (string, bool) {
	select {
	case r.forwards <- struct{}{}:
	default:
		metrics.ShedTotal.WithLabelValues("forwards").Inc()
		return "forwards", false
	}

	r.workspaceMu.Lock()
	defer r.workspaceMu.Unlock()
	limit, metric := r.config.ForwardsPerWorkspace, "forwards_per_workspace"
	if workspace == "" {
		limit, metric = r.config.ForwardsUnidentified, "forwards_unidentified"
	}
	if r.workspaceForwards[workspace] >= limit {
		<-r.forwards
		metrics.ShedTotal.WithLabelValues(metric).Inc()
		return metric, false
	}
	r.workspaceForwards[workspace]++
	return "", true
}

func (r *Resolver) releaseForward(workspace string) {
	r.workspaceMu.Lock()
	if r.workspaceForwards[workspace] <= 1 {
		delete(r.workspaceForwards, workspace)
	} else {
		r.workspaceForwards[workspace]--
	}
	r.workspaceMu.Unlock()
	<-r.forwards
}

func (r *Resolver) forward(ctx context.Context, request *dnswire.Msg, transport string, identity discovery.Caller) (*dnswire.Msg, outcome) {
	failure := func(why reason, err error) (*dnswire.Msg, outcome) {
		response := new(dnswire.Msg)
		dnsutil.SetReply(response, request)
		response.Rcode = dnswire.RcodeServerFailure
		response.RecursionAvailable = true
		return response, outcome{reason: why, err: err, caller: identity}
	}

	if limit, ok := r.acquireForward(identity.Workspace); !ok {
		return failure(reasonShed, fmt.Errorf("%s concurrency limit is full", limit))
	}
	defer r.releaseForward(identity.Workspace)

	query := dnswire.NewMsg(request.Question[0].Header().Name, dnswire.RRToType(request.Question[0]), request.Question[0].Header().Class)
	query.RecursionDesired = request.RecursionDesired
	query.UDPSize = 1232
	if response := r.cache.get(ctx, query, false); response != nil {
		response.ID = request.ID
		response.Question = copyQuestion(request.Question[0])
		return response, outcome{reason: reasonUpstream, err: nil, caller: identity}
	}

	exchangeCtx, cancel := context.WithTimeout(ctx, r.config.ForwardTimeout)
	defer cancel()
	result := r.exchange(exchangeCtx, query, transport)
	if result.err != nil {
		if ctx.Err() == nil && (result.reason == reasonUpstreamError || result.reason == reasonUpstreamTimeout) {
			if response := r.cache.get(ctx, query, true); response != nil {
				response.ID = request.ID
				response.Question = copyQuestion(request.Question[0])
				return response, outcome{reason: reasonCacheStale, err: nil, caller: identity}
			}
		}
		return failure(result.reason, result.err)
	}

	response := result.response
	response.Question = copyQuestion(request.Question[0])
	zeroOutOfRangeTTLs(response)
	if response.Rcode == dnswire.RcodeServerFailure {
		metrics.UpstreamHealthy.Set(0)
		if ctx.Err() == nil {
			if stale := r.cache.get(ctx, query, true); stale != nil {
				stale.ID = request.ID
				stale.Question = copyQuestion(request.Question[0])
				return stale, outcome{reason: reasonCacheStale, err: nil, caller: identity}
			}
		}
		response.ID = request.ID
		return response, outcome{reason: reasonUpstream, err: nil, caller: identity}
	}

	if err := r.cache.set(ctx, query, response); err != nil {
		logger.Warn("cache upstream DNS response", "error_type", fmt.Sprintf("%T", err))
	}
	response.ID = request.ID
	return response, outcome{reason: reasonUpstream, err: nil, caller: identity}
}

func (r *Resolver) exchange(ctx context.Context, query *dnswire.Msg, transport string) forwardResult {
	response, _, err := r.client.Exchange(ctx, query, transport, r.config.Upstream)
	if err == nil && response.Truncated && transport == "udp" {
		query.Data = nil
		response, _, err = r.client.Exchange(ctx, query, "tcp", r.config.Upstream)
	}
	if err != nil {
		metrics.UpstreamHealthy.Set(0)
		why := reasonUpstreamError
		if netErr, ok := errors.AsType[net.Error](err); ok && netErr.Timeout() || errors.Is(err, context.DeadlineExceeded) {
			why = reasonUpstreamTimeout
		}
		return forwardResult{response: nil, reason: why, err: err}
	}

	if response == nil || !response.Response || response.Opcode != dnswire.OpcodeQuery ||
		len(response.Question) != 1 || !sameQuestion(response.Question[0], query.Question[0]) {
		metrics.UpstreamHealthy.Set(0)
		return forwardResult{response: nil, reason: reasonUpstreamInvalid, err: errors.New("upstream response does not answer the forwarded question")}
	}
	metrics.UpstreamHealthy.Set(1)
	return forwardResult{response: response, reason: reasonUpstream, err: nil}
}

func copyQuestion(question dnswire.RR) []dnswire.RR {
	copy := dnswire.NewMsg(question.Header().Name, dnswire.RRToType(question), question.Header().Class)
	return copy.Question
}

func sameQuestion(a, b dnswire.RR) bool {
	return a.Header().Name == b.Header().Name && a.Header().Class == b.Header().Class && dnswire.RRToType(a) == dnswire.RRToType(b)
}

// RFC 2181 section 8 requires TTLs with the most significant bit set to be
// treated as zero. An OPT record's TTL field carries EDNS flags, not a TTL.
func zeroOutOfRangeTTLs(response *dnswire.Msg) {
	for _, section := range [][]dnswire.RR{response.Answer, response.Ns, response.Extra} {
		for _, record := range section {
			if _, pseudo := record.(*dnswire.OPT); pseudo {
				continue
			}
			if record.Header().TTL > math.MaxInt32 {
				record.Header().TTL = 0
			}
			if soa, ok := record.(*dnswire.SOA); ok && soa.Minttl > math.MaxInt32 {
				soa.Minttl = 0
			}
		}
	}
}
