package resolver

import (
	"errors"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/svc/undns/internal/discovery"
)

type reason string

const (
	reasonAnswer              reason = "answer"
	reasonNoData              reason = "nodata"
	reasonUnknownName         reason = "unknown_name"
	reasonUnsupportedType     reason = "unsupported_type"
	reasonIdentityUnavailable reason = "identity_unavailable"
	reasonDiscoveryNotReady   reason = "discovery_not_ready"
	reasonUpstream            reason = "upstream"
	reasonUpstreamTimeout     reason = "upstream_timeout"
	reasonUpstreamError       reason = "upstream_error"
	reasonUpstreamInvalid     reason = "upstream_invalid"
	reasonCacheStale          reason = "cache_stale"
	reasonShed                reason = "shed"
	reasonMalformed           reason = "malformed"
	reasonUnsupportedOpcode   reason = "unsupported_opcode"
	reasonPackError           reason = "pack_error"
)

var errUnknownSource error = &discovery.Error{Reason: discovery.ReasonUnknownCaller, Err: errors.New("source address is not an IP")}

func failureReason(err error) reason {
	return reason(discovery.ReasonOf(err))
}

type outcome struct {
	reason reason
	err    error
	caller discovery.Caller
}

func noCaller() discovery.Caller {
	return discovery.Caller{Workspace: "", Project: "", Deployment: "", Namespace: ""}
}

type sampler struct {
	mu       sync.Mutex
	last     map[reason]time.Time
	interval time.Duration
	clock    clock.Clock
}

func newSampler(interval time.Duration, clk clock.Clock) *sampler {
	return &sampler{mu: sync.Mutex{}, last: make(map[reason]time.Time), interval: interval, clock: clk}
}

func (s *sampler) allow(key reason) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.clock.Now()
	if last, ok := s.last[key]; ok && now.Sub(last) < s.interval {
		return false
	}
	s.last[key] = now
	return true
}
