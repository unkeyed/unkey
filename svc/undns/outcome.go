package undns

import (
	"errors"
	"sync"
	"time"
)

type reason string

const (
	reasonAnswer              reason = "answer"
	reasonNoData              reason = "nodata"
	reasonUnknownName         reason = "unknown_name"
	reasonUnsupportedType     reason = "unsupported_type"
	reasonUnknownCaller       reason = "unknown_caller"
	reasonAmbiguousCaller     reason = "ambiguous_caller"
	reasonIneligibleCaller    reason = "ineligible_caller"
	reasonIdentityUnavailable reason = "identity_unavailable"
	reasonDiscoveryNotReady   reason = "discovery_not_ready"
	reasonBindingUnresolved   reason = "binding_unresolved"
	reasonBindingAmbiguous    reason = "binding_ambiguous"
	reasonBindingInvalid      reason = "binding_invalid"
	reasonServiceMissing      reason = "service_missing"
	reasonServiceRejected     reason = "service_rejected"
	reasonServiceRetired      reason = "service_retired"
	reasonNoReadyEndpoints    reason = "no_ready_endpoints"
	reasonLookupError         reason = "lookup_error"
	reasonUpstream            reason = "upstream"
	reasonUpstreamTimeout     reason = "upstream_timeout"
	reasonUpstreamError       reason = "upstream_error"
	reasonUpstreamInvalid     reason = "upstream_invalid"
	reasonShed                reason = "shed"
	reasonMalformed           reason = "malformed"
	reasonUnsupportedOpcode   reason = "unsupported_opcode"
	reasonPackError           reason = "pack_error"

	stateActive          reason = "active"
	statePendingRevision reason = "pending_revision"
)

var (
	errUnknownCaller     = errors.New("no pod uses the source IP")
	errAmbiguousCaller   = errors.New("several pods use the source IP")
	errIneligibleCaller  = errors.New("source pod is not a running Krane deployment pod")
	errBindingAmbiguous  = errors.New("ambiguous app binding")
	errBindingUnresolved = errors.New("binding target is unresolved")
	errBindingInvalid    = errors.New("invalid app binding")
	errServiceMissing    = errors.New("selected discovery service is unavailable")
	errServiceRejected   = errors.New("selected discovery service does not match binding")
	errServiceRetired    = errors.New("discovery replacement overlap expired")
	errNoReadyEndpoints  = errors.New("selected deployment has no ready endpoints")
)

func failureReason(err error) reason {
	switch {
	case errors.Is(err, errUnknownCaller):
		return reasonUnknownCaller
	case errors.Is(err, errAmbiguousCaller):
		return reasonAmbiguousCaller
	case errors.Is(err, errIneligibleCaller):
		return reasonIneligibleCaller
	case errors.Is(err, errBindingAmbiguous):
		return reasonBindingAmbiguous
	case errors.Is(err, errBindingUnresolved):
		return reasonBindingUnresolved
	case errors.Is(err, errBindingInvalid):
		return reasonBindingInvalid
	case errors.Is(err, errServiceMissing):
		return reasonServiceMissing
	case errors.Is(err, errServiceRejected):
		return reasonServiceRejected
	case errors.Is(err, errServiceRetired):
		return reasonServiceRetired
	case errors.Is(err, errNoReadyEndpoints):
		return reasonNoReadyEndpoints
	default:
		return reasonLookupError
	}
}

type outcome struct {
	reason reason
	err    error
	caller caller
}

type sampler struct {
	mu       sync.Mutex
	last     map[reason]time.Time
	interval time.Duration
	now      func() time.Time
}

func newSampler(interval time.Duration) *sampler {
	return &sampler{mu: sync.Mutex{}, last: make(map[reason]time.Time), interval: interval, now: time.Now}
}

func (s *sampler) allow(key reason) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	if last, ok := s.last[key]; ok && now.Sub(last) < s.interval {
		return false
	}
	s.last[key] = now
	return true
}
