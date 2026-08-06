// Package domaingate holds the custom domain preconditions shared by the two
// layers that each enforce them: the public API handler and the ctrl RPC service.
// Centralizing the invariant here is what stops the two copies from drifting — a
// domain the API accepts is a domain ctrl accepts, by construction.
//
// The Check* functions return nil when the action is allowed, or a fault carrying
// the error code and a caller-facing message. Surface that message with
// fault.UserFacingMessage(err) — never err.Error(), which also includes internal
// detail. The API returns the fault directly (its error middleware renders
// UserFacingMessage); ctrl wraps UserFacingMessage into its connect error.
package domaingate

import (
	"fmt"
	"strings"

	"golang.org/x/net/idna"
	"golang.org/x/net/publicsuffix"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// hostnameProfile is the IDNA profile domains are parsed under. MapForLookup
// deliberately omits the bidirectional-text rule, hence BidiRule on top, and
// VerifyDNSLength enforces the RFC 1035 caps of 63 octets per label and 253
// for the whole name.
var hostnameProfile = idna.New(
	idna.MapForLookup(),
	idna.BidiRule(),
	idna.VerifyDNSLength(true),
)

// ParseDomain validates input as a name Unkey can attach and returns its
// canonical form: lowercase ASCII, with Unicode labels Punycode encoded, which
// is the form DNS and TLS issuance operate on. Both layers persist and compare
// the canonical form, so 'münchen.de' and 'xn--mnchen-3ya.de' are one domain.
//
// This accepts fewer values than DNS itself: schemes, ports, paths, IP-shaped
// names, wildcards, and trailing root dots are rejected, and the name must sit
// under a registrable public-suffix domain — 'co.uk' is a valid DNS name, but
// nobody can own it, so verification could never succeed. A successful parse
// proves shape, not ownership.
func ParseDomain(input string) (string, error) {
	if input == "" {
		return "", invalidDomain(input, "hostname is empty")
	}
	if strings.TrimSpace(input) != input {
		return "", invalidDomain(input, "surrounding whitespace is not allowed")
	}
	if strings.HasSuffix(input, ".") {
		return "", invalidDomain(input, "trailing root dot is not allowed")
	}
	if strings.Contains(input, "*") {
		return "", invalidDomain(input, "wildcards are not allowed")
	}

	hostname, err := hostnameProfile.ToASCII(input)
	if err != nil {
		return "", invalidDomain(input, fmt.Sprintf("IDNA validation failed: %v", err))
	}

	// IDNA maps several Unicode full-stop characters to the label separator, so
	// the canonical value can gain a trailing dot the raw input never had.
	if strings.HasSuffix(hostname, ".") {
		return "", invalidDomain(input, "trailing root dot is not allowed")
	}

	labels := strings.Split(hostname, ".")
	finalLabel := labels[len(labels)-1]
	if looksLikeIPv4Label(finalLabel) {
		return "", invalidDomain(input, "hostname resembles an IP address")
	}

	// Real TLDs are at least two characters. The public-suffix list falls back to
	// treating an unknown TLD as a suffix, which would wave 'api.acme.c' through.
	if len(finalLabel) < 2 {
		return "", invalidDomain(input, "top-level domain is too short")
	}

	if _, err := publicsuffix.EffectiveTLDPlusOne(hostname); err != nil {
		return "", fault.New("domain is not registrable",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal(fmt.Sprintf("domain %q has no registrable domain: %v", input, err)),
			fault.Public(fmt.Sprintf("The domain '%s' is a public suffix that nobody can own. Pass a domain registered to you, such as 'api.acme.com'.", input)),
		)
	}

	return hostname, nil
}

// AlreadyExists is the outcome for a domain the workspace already holds. Domains
// are unique per workspace, so the same name cannot serve two environments.
//
// Scoped to the workspace, not to Unkey: the unique index is (workspace_id, domain), and
// a name another workspace holds can still be claimed here by proving ownership.
func AlreadyExists(domain string) error {
	return fault.New("domain already exists",
		fault.Code(codes.Data.Domain.Duplicate.URN()),
		fault.Internal(fmt.Sprintf("domain %q is already attached to this workspace", domain)),
		fault.Public(fmt.Sprintf("The domain '%s' is already attached to this workspace.", domain)),
	)
}

// AlreadyVerified is the outcome when a caller retries verification of a domain
// that already passed. A verified domain serves traffic. A second run of the
// success path (frontline route creation, contested-domain revocation) against
// live state can break it, thus this state is terminal for this operation.
func AlreadyVerified(domain string) error {
	return fault.New("domain already verified",
		fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
		fault.Internal(fmt.Sprintf("domain %q is already verified", domain)),
		fault.Public(fmt.Sprintf("The domain '%s' is already verified. No action is needed.", domain)),
	)
}

// CheckAllowance reports whether the workspace may attach one more domain. The
// counts stay internal: the way out is the same either way.
func CheckAllowance(attached int64, allowed uint32) error {
	if attached < int64(allowed) {
		return nil
	}

	return fault.New("custom domain allowance reached",
		fault.Code(codes.Limits.CustomDomain.Exceeded.URN()),
		fault.Internal(fmt.Sprintf("workspace holds %d of %d allowed custom domains", attached, allowed)),
		fault.Public("Your plan does not allow another custom domain. Upgrade your plan, or remove a domain you no longer need, then retry."),
	)
}

// LimitsNotConfigured is the outcome for a workspace with no limits row. Billing
// writes every allowance, so a missing row means unknown billing state, not free
// tier. The caller cannot fix it, hence support rather than an upgrade.
func LimitsNotConfigured(workspaceID string) error {
	return fault.New("workspace limits not configured",
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Internal(fmt.Sprintf("workspace %q has no limits row", workspaceID)),
		fault.Public("Resource limits are not configured for this workspace. Contact support@unkey.com."),
	)
}

func invalidDomain(domain, reason string) error {
	return fault.New("invalid domain",
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Internal(fmt.Sprintf("domain %q rejected: %s", domain, reason)),
		fault.Public(fmt.Sprintf("The domain '%s' is not a valid fully qualified domain name. Pass a name such as 'api.acme.com', without a scheme, port, or path.", domain)),
	)
}

// looksLikeIPv4Label reports whether the final label is decimal or hexadecimal,
// the shapes WHATWG URL parsing treats as IPv4-like. Rejecting them prevents
// values such as 012.0.0.1 and 0x7f.0.0.1 from resolving differently across
// clients. The label is already lowercased by the IDNA mapping, so only
// lowercase hex needs recognizing.
func looksLikeIPv4Label(label string) bool {
	decimal := true
	for i := 0; i < len(label); i++ {
		if label[i] < '0' || label[i] > '9' {
			decimal = false
			break
		}
	}
	if decimal {
		return true
	}

	if len(label) <= 2 || label[0] != '0' || label[1] != 'x' {
		return false
	}
	for i := 2; i < len(label); i++ {
		c := label[i]
		isHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')
		if !isHex {
			return false
		}
	}

	return true
}
