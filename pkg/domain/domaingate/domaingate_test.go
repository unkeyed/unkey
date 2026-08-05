package domaingate_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/domain/domaingate"
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

// requireParseRejects asserts ParseDomain rejects input with the validation code.
func requireParseRejects(t *testing.T, input string) {
	t.Helper()
	_, err := domaingate.ParseDomain(input)
	requireCode(t, codes.App.Validation.InvalidInput, err)
}

func TestParseDomainCanonicalizes(t *testing.T) {
	t.Parallel()

	// input → canonical form. Persisting the canonical form is what makes
	// 'MÜNCHEN.DE' and 'xn--mnchen-3ya.de' collide on the unique index instead of
	// coexisting as two rows for one name.
	for input, want := range map[string]string{
		"api.acme.com":        "api.acme.com",
		"API.ACME.COM":        "api.acme.com",
		"acme.com":            "acme.com",
		"a-b.c-d.example.com": "a-b.c-d.example.com",
		"münchen.de":          "xn--mnchen-3ya.de",
		"MÜNCHEN.DE":          "xn--mnchen-3ya.de",
		"xn--mnchen-3ya.de":   "xn--mnchen-3ya.de",
		"日本.jp":               "xn--wgv71a.jp",
		// Registrable multi-part suffixes: the name under them is fine, the
		// suffix itself is not (covered in TestParseDomainRejectsPublicSuffixes).
		"acme.co.uk":     "acme.co.uk",
		"api.acme.co.uk": "api.acme.co.uk",
		// github.io sits on the public suffix list's private section, so a
		// user site under it is registrable in PSL terms.
		"acme.github.io": "acme.github.io",
		// Unknown TLD: the PSL default rule makes the TLD itself the suffix, so
		// anything one level below it passes.
		"acme.notarealtld": "acme.notarealtld",
	} {
		got, err := domaingate.ParseDomain(input)
		require.NoError(t, err, "input %q", input)
		require.Equal(t, want, got, "input %q", input)
	}

	// 63-octet label and 253-octet total are the RFC 1035 caps, enforced by the
	// IDNA profile rather than by counting here.
	require.NotPanics(t, func() {
		longestLabel := strings.Repeat("k", 63) + ".acme.com"
		got, err := domaingate.ParseDomain(longestLabel)
		require.NoError(t, err)
		require.Equal(t, longestLabel, got)
	})
	label := strings.Repeat("k", 49)
	longest := strings.Join([]string{label, label, label, label, label}, ".") + ".com"
	require.Len(t, longest, 253)
	_, err := domaingate.ParseDomain(longest)
	require.NoError(t, err)
}

func TestParseDomainRejectsMalformed(t *testing.T) {
	t.Parallel()

	for _, invalid := range []string{
		"",
		"localhost",
		".acme.com",
		"acme.com.",
		"a..com",
		"-api.acme.com",
		"api-.acme.com",
		"api_v2.acme.com",
		"api.acme.c",
		"https://api.acme.com",
		"api.acme.com/v1",
		"api.acme.com:8080",
		"api acme.com",
		" api.acme.com",
		"api.acme.com ",
		"*.acme.com",
		"xn--a.com",
		strings.Repeat("a", 250) + ".com",
		strings.Repeat("kebap", 13) + ".acme.com",
		"acme." + strings.Repeat("k", 64),
	} {
		requireParseRejects(t, invalid)
	}

	// IDNA maps U+3002 (ideographic full stop) to a label separator, so this only
	// gains its trailing dot after canonicalization. The raw-input check alone
	// would wave it through.
	requireParseRejects(t, "acme.com。")

	// One octet past the 253-octet cap, with every label individually valid.
	label := strings.Repeat("k", 49)
	longest := strings.Join([]string{label, label, label, label, label}, ".") + ".com"
	requireParseRejects(t, "k"+longest)
}

// WHATWG URL parsing treats a decimal or hexadecimal final label as IPv4-like,
// and such values resolve differently across clients.
func TestParseDomainRejectsIPLikeNames(t *testing.T) {
	t.Parallel()

	for _, invalid := range []string{
		"1.2.3.4",
		"127.0.0.1",
		"012.0.0.1",
		"0x7f.0.0.1",
		"acme.0x7f",
		"acme.1",
	} {
		requireParseRejects(t, invalid)
	}
}

// A public suffix is a valid DNS name nobody can register, so verification
// could never succeed against it.
func TestParseDomainRejectsPublicSuffixes(t *testing.T) {
	t.Parallel()

	for _, suffix := range []string{
		"com",
		"uk",
		"co.uk",
		"github.io",
	} {
		requireParseRejects(t, suffix)
	}

	_, err := domaingate.ParseDomain("co.uk")
	require.Equal(t,
		"The domain 'co.uk' is a public suffix that nobody can own. Pass a domain registered to you, such as 'api.acme.com'.",
		fault.UserFacingMessage(err))
}

func TestAlreadyExists(t *testing.T) {
	t.Parallel()

	requireCode(t, codes.Data.Domain.Duplicate, domaingate.AlreadyExists("api.acme.com"))
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

	_, parseErr := domaingate.ParseDomain("bad domain")
	require.Equal(t,
		"The domain 'bad domain' is not a valid fully qualified domain name. Pass a name such as 'api.acme.com', without a scheme, port, or path.",
		fault.UserFacingMessage(parseErr))

	require.Equal(t,
		"The domain 'api.acme.com' is already attached to this workspace.",
		fault.UserFacingMessage(domaingate.AlreadyExists("api.acme.com")))

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
