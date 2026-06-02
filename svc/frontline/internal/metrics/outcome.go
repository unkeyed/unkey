package metrics

import "github.com/unkeyed/unkey/pkg/codes"

// Outcome classifies a frontline request by where in the request path
// it ended. Derived from the fault URN via urnOutcomes and emitted as
// a label on RequestsTotal.
type Outcome string

const (
	// OutcomeSuccess — request completed without a fault URN.
	OutcomeSuccess Outcome = "success"

	// OutcomeRefused — frontline returned an error based on the
	// request or route state before reaching an upstream: auth
	// rejected, rate-limited, request invalid, no matching route for
	// the hostname.
	OutcomeRefused Outcome = "refused"

	// OutcomeFrontlineFault — failure inside frontline itself: a 5xx
	// we threw, a config/DB lookup that returned an error, our
	// gateway deadline expiring, an unclassified proxy error, or any
	// failure on the peer-frontline hop (DNS, dial, reset, timeout).
	OutcomeFrontlineFault Outcome = "frontline_fault"

	// OutcomeUpstreamProblem — failure reaching or talking to a
	// customer instance: connection refused, host unreachable, reset,
	// dial or response timeout, or no instance available to dial. The
	// URN carries the specific mechanism.
	OutcomeUpstreamProblem Outcome = "upstream_problem"

	// OutcomeNoise — no-route hit on a hostname not tied to a known
	// customer route.
	OutcomeNoise Outcome = "noise"
)

// urnOutcomes maps every frontline URN to an Outcome. outcome_test.go
// fails CI if a frontline URN is missing here. OutcomeFor defaults
// unknown URNs to OutcomeFrontlineFault so a missing entry surfaces as
// a frontline fault rather than as silence.
var urnOutcomes = map[codes.URN]Outcome{
	// ── frontline_fault: frontline-internal or peer-frontline path ──
	codes.Frontline.Internal.InternalServerError.URN():         OutcomeFrontlineFault,
	codes.Frontline.Internal.ConfigLoadFailed.URN():            OutcomeFrontlineFault,
	codes.Frontline.Internal.InstanceLoadFailed.URN():          OutcomeFrontlineFault,
	codes.Frontline.Internal.InvalidConfiguration.URN():        OutcomeFrontlineFault,
	codes.Frontline.Routing.DeploymentSelectionFailed.URN():    OutcomeFrontlineFault,
	codes.Frontline.Routing.NoReachableRegion.URN():            OutcomeFrontlineFault,
	codes.Frontline.Proxy.GatewayDeadlineExceeded.URN():        OutcomeFrontlineFault,
	codes.Frontline.Proxy.ProxyErrorUnclassified.URN():         OutcomeFrontlineFault,
	codes.Frontline.Proxy.PeerFrontlineConnectionRefused.URN(): OutcomeFrontlineFault,
	codes.Frontline.Proxy.PeerFrontlineHostUnreachable.URN():   OutcomeFrontlineFault,
	codes.Frontline.Proxy.PeerFrontlineConnectionReset.URN():   OutcomeFrontlineFault,
	codes.Frontline.Proxy.PeerFrontlineDNSNotFound.URN():       OutcomeFrontlineFault,
	codes.Frontline.Proxy.PeerFrontlineDNSTimeout.URN():        OutcomeFrontlineFault,
	codes.Frontline.Proxy.PeerFrontlineTimeout.URN():           OutcomeFrontlineFault,

	// ── upstream_problem: reaching/talking to a customer instance ──
	codes.Frontline.Proxy.UpstreamConnectionRefused.URN(): OutcomeUpstreamProblem,
	codes.Frontline.Proxy.UpstreamHostUnreachable.URN():   OutcomeUpstreamProblem,
	codes.Frontline.Proxy.UpstreamConnectionReset.URN():   OutcomeUpstreamProblem,
	codes.Frontline.Proxy.DialTimeout.URN():               OutcomeUpstreamProblem,
	codes.Frontline.Proxy.UpstreamResponseTimeout.URN():   OutcomeUpstreamProblem,
	codes.Frontline.Routing.NoDeploymentInstances.URN():   OutcomeUpstreamProblem,
	codes.Frontline.Routing.NoRunningInstances.URN():      OutcomeUpstreamProblem,

	// ── refused: request rejected on its own merits ────────────────
	codes.Frontline.Routing.ConfigNotFoundForCustomDomain.URN(): OutcomeRefused,
	codes.Frontline.Routing.DeploymentNotFound.URN():            OutcomeRefused,
	codes.Frontline.Auth.MissingCredentials.URN():               OutcomeRefused,
	codes.Frontline.Auth.InvalidKey.URN():                       OutcomeRefused,
	codes.Frontline.Auth.InsufficientPermissions.URN():          OutcomeRefused,
	codes.Frontline.Auth.RateLimited.URN():                      OutcomeRefused,
	codes.Frontline.Firewall.Denied.URN():                       OutcomeRefused,
	codes.Frontline.OpenApi.InvalidRequest.URN():                OutcomeRefused,
	// 4xx URNs originating outside frontline that we surface as-is —
	// listed here so OutcomeFor doesn't default them to frontline_fault.
	codes.User.BadRequest.ClientClosedRequest.URN():   OutcomeRefused,
	codes.User.BadRequest.RequestTimeout.URN():        OutcomeRefused,
	codes.User.BadRequest.RequestBodyTooLarge.URN():   OutcomeRefused,
	codes.User.BadRequest.RequestBodyUnreadable.URN(): OutcomeRefused,
	codes.Auth.Authentication.Missing.URN():           OutcomeRefused,
	codes.Auth.Authentication.Malformed.URN():         OutcomeRefused,
	codes.App.Validation.InvalidInput.URN():           OutcomeRefused,

	// ── noise: no-route on anonymously reachable hostname ──────────
	codes.Frontline.Routing.ConfigNotFoundForUnkeyHostname.URN(): OutcomeNoise,
}

// OutcomeFor returns the Outcome for a given URN. Empty URN is
// OutcomeSuccess. Unknown URNs default to OutcomeFrontlineFault so a
// missing entry surfaces as a frontline fault rather than as silence.
func OutcomeFor(urn codes.URN) Outcome {
	if urn == "" {
		return OutcomeSuccess
	}
	if o, ok := urnOutcomes[urn]; ok {
		return o
	}
	return OutcomeFrontlineFault
}
