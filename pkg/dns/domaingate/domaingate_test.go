package domaingate_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/dns"
	"github.com/unkeyed/unkey/pkg/dns/domaingate"
	"github.com/unkeyed/unkey/pkg/fault"
)

// requireCode asserts err is a fault carrying the given code.
func requireCode(t *testing.T, want codes.Code, err error) {
	t.Helper()
	require.Error(t, err)
	got, ok := fault.GetCode(err)
	require.True(t, ok, "expected a coded fault")
	require.Equal(t, want.URN(), got)
}

func TestCheckDomain(t *testing.T) {
	t.Parallel()

	require.NoError(t, domaingate.CheckDomain("api.acme.com"))
	require.NoError(t, domaingate.CheckDomain("acme.com"))
	require.NoError(t, domaingate.CheckDomain("a-b.c-d.example.com"))
	require.NoError(t, domaingate.CheckDomain(strings.Repeat("k", 63)+".acme.com"))

	// The total-length cap is the only thing rejecting this: every label is well
	// under 63, so the pattern alone would accept it. Built to land exactly on
	// MaxFQDNLength and then one octet past it.
	label := strings.Repeat("k", 49)
	longest := strings.Join([]string{label, label, label, label, label}, ".") + ".com"
	require.Len(t, longest, dns.MaxFQDNLength)
	require.NoError(t, domaingate.CheckDomain(longest))
	requireCode(t, codes.App.Validation.InvalidInput, domaingate.CheckDomain("k"+longest))

	for _, invalid := range []string{
		"",
		"localhost",
		".acme.com",
		"acme.com.",
		"-api.acme.com",
		"api-.acme.com",
		"api_v2.acme.com",
		"api.acme.c",
		"https://api.acme.com",
		"api.acme.com/v1",
		"api.acme.com:8080",
		"api acme.com",
		"*.acme.com",
		strings.Repeat("a", 250) + ".com",
		// Each of these fits inside MaxFQDNLength, so only the per-label cap
		// rejects them. Total length alone would let all three through.
		strings.Repeat("kebap", 13) + ".acme.com",
		"api." + strings.Repeat("kebap", 13) + ".com",
		"acme." + strings.Repeat("k", 64),
	} {
		requireCode(t, codes.App.Validation.InvalidInput, domaingate.CheckDomain(invalid))
	}
}

func TestAlreadyAttached(t *testing.T) {
	t.Parallel()

	requireCode(t, codes.Data.Domain.Duplicate, domaingate.AlreadyAttached("api.acme.com"))
}

// Both layers pass the outcome of their own lookup, so a nil error meaning "free" and a
// found row meaning "taken" is decided here rather than at each call site.
func TestCheckNotAttached(t *testing.T) {
	t.Parallel()

	require.NoError(t, domaingate.CheckNotAttached("api.acme.com", false))
	requireCode(t, codes.Data.Domain.Duplicate, domaingate.CheckNotAttached("api.acme.com", true))
}

func TestCheckAllowance(t *testing.T) {
	t.Parallel()

	require.NoError(t, domaingate.CheckAllowance(0, 1))
	require.NoError(t, domaingate.CheckAllowance(4, 5))

	// At the allowance, not merely over it.
	requireCode(t, codes.Limits.CustomDomain.Exceeded, domaingate.CheckAllowance(1, 1))

	// A zero allowance refuses the first domain, which is the free tier.
	requireCode(t, codes.Limits.CustomDomain.Exceeded, domaingate.CheckAllowance(0, 0))

	// An allowance already overshot by a concurrent create stays closed.
	requireCode(t, codes.Limits.CustomDomain.Exceeded, domaingate.CheckAllowance(3, 1))
}

// ctrl surfaces fault.UserFacingMessage on its own error surface, so verify each
// outcome exposes exactly the caller-facing message and never the internal detail.
func TestFaultUserFacingMessage(t *testing.T) {
	t.Parallel()

	require.Equal(t,
		"The domain 'bad domain' is not a valid fully qualified domain name. Pass a name such as 'api.acme.com', without a scheme, port, or path.",
		fault.UserFacingMessage(domaingate.CheckDomain("bad domain")))

	require.Equal(t,
		"The domain 'api.acme.com' is already attached to this workspace.",
		fault.UserFacingMessage(domaingate.AlreadyAttached("api.acme.com")))

	require.Equal(t,
		"Your plan does not allow another custom domain. Upgrade your plan, or remove a domain you no longer need, then retry.",
		fault.UserFacingMessage(domaingate.CheckAllowance(1, 1)))

	require.Equal(t,
		"Resource limits are not configured for this workspace. Contact support@unkey.com.",
		fault.UserFacingMessage(domaingate.LimitsNotConfigured("ws_1234abcd")))

	// The allowance counts and the workspace id describe billing state the caller
	// cannot act on, so they stay internal.
	require.NotContains(t, fault.UserFacingMessage(domaingate.CheckAllowance(7, 2)), "7")
	require.NotContains(t, fault.UserFacingMessage(domaingate.LimitsNotConfigured("ws_1234abcd")), "ws_1234abcd")
}
