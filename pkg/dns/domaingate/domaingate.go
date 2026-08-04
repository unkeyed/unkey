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

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/dns"
	"github.com/unkeyed/unkey/pkg/fault"
)

// CheckDomain reports whether domain is a name Unkey can attach at all. The API
// advertises the same rule as its `domain` request pattern, so a caller reaching
// this check bypassed the schema.
func CheckDomain(domain string) error {
	if dns.IsValidFQDN(domain) {
		return nil
	}

	return fault.New("invalid domain",
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Internal(fmt.Sprintf("domain %q does not match the FQDN pattern", domain)),
		fault.Public(fmt.Sprintf("The domain '%s' is not a valid fully qualified domain name. Pass a name such as 'api.acme.com', without a scheme, port, or path.", domain)),
	)
}

// CheckAvailable reports whether domain is free to attach within the workspace.
func CheckAvailable(domain string, taken bool) error {
	if !taken {
		return nil
	}

	return fault.New("domain already exists",
		fault.Code(codes.Data.Domain.Duplicate.URN()),
		fault.Internal(fmt.Sprintf("domain %q is already registered in this workspace", domain)),
		fault.Public(fmt.Sprintf("The domain '%s' is already registered in this workspace.", domain)),
	)
}

// CheckAllowance reports whether the workspace may attach one more domain.
//
// The counts stay internal: they describe billing state the caller cannot act on,
// and the way out is the same either way.
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
// writes every allowance, so a missing row means billing state is unknown rather
// than that the workspace is on the free tier. Refusing is the safe reading, but
// the caller cannot fix it, so the message points at support.
func LimitsNotConfigured(workspaceID string) error {
	return fault.New("workspace limits not configured",
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Internal(fmt.Sprintf("workspace %q has no limits row", workspaceID)),
		fault.Public("Resource limits are not configured for this workspace. Contact support@unkey.com."),
	)
}
