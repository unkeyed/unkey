package domaingate_test

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/domain/domaingate"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
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

// The lookup handlers accept one field that holds either a name or an id, and they
// tell the two apart by whether ParseDomain accepts the value. Pin both halves: a
// name arrives canonical, and everything else arrives untouched for the id column.
func TestCanonicalizeIdentifier(t *testing.T) {
	t.Parallel()

	for input, want := range map[string]string{
		"api.acme.com": "api.acme.com",
		"MÜNCHEN.DE":   "xn--mnchen-3ya.de",
		"münchen.de":   "xn--mnchen-3ya.de",
	} {
		require.Equal(t, want, domaingate.CanonicalizeIdentifier(input), "input %q", input)
	}

	// An id contains an underscore, which is not a legal hostname rune, so no id can
	// reach the canonicalizing branch. Rejection is what routes it to the id column.
	id := uid.New(uid.DomainPrefix)
	requireParseRejects(t, id)
	require.Equal(t, id, domaingate.CanonicalizeIdentifier(id))

	// Values that are neither a name nor an id also pass through. The caller reports
	// the lookup miss, so this function has no rejection of its own.
	for _, input := range []string{
		"KEBAP",
		"co.uk",
		"*.acme.com",
		"api.acme.com/nope",
		"api.acme.com:443",
		"",
	} {
		require.Equal(t, input, domaingate.CanonicalizeIdentifier(input), "input %q", input)
	}
}

func TestAlreadyExists(t *testing.T) {
	t.Parallel()

	requireCode(t, codes.Data.Domain.Duplicate, domaingate.AlreadyExists("api.acme.com"))
}

func TestCheckAllowance(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		attached []string
		domain   string
		allowed  uint32
		accepted bool
	}{
		{name: "first domain under the allowance", attached: nil, domain: "api.acme.com", allowed: 1, accepted: true},
		{name: "new registrable domain with a free slot", attached: []string{"api.acme.com"}, domain: "api.other.com", allowed: 2, accepted: true},
		{name: "new registrable domain at the allowance", attached: []string{"api.acme.com"}, domain: "api.other.com", allowed: 1, accepted: false},
		{name: "sibling subdomain at the allowance", attached: []string{"api.acme.com"}, domain: "app.acme.com", allowed: 1, accepted: true},
		{name: "registrable domain itself at the allowance", attached: []string{"api.acme.com"}, domain: "acme.com", allowed: 1, accepted: true},
		{name: "nested subdomain at the allowance", attached: []string{"api.acme.com"}, domain: "v1.api.acme.com", allowed: 1, accepted: true},
		{name: "subdomain of the held registrable domain", attached: []string{"acme.com"}, domain: "api.acme.com", allowed: 1, accepted: true},
		{name: "zero allowance refuses the first domain", attached: nil, domain: "api.acme.com", allowed: 0, accepted: false},
		{name: "zero allowance refuses a subdomain of a kept domain", attached: []string{"api.acme.com"}, domain: "app.acme.com", allowed: 0, accepted: false},
		{name: "multi-label suffix shares the registrable domain", attached: []string{"a.acme.co.uk"}, domain: "b.acme.co.uk", allowed: 1, accepted: true},
		{name: "multi-label suffix separates registrable domains", attached: []string{"acme.co.uk"}, domain: "other.co.uk", allowed: 1, accepted: false},
		{name: "private suffix separates its users", attached: []string{"a.github.io"}, domain: "b.github.io", allowed: 1, accepted: false},
		{name: "overshot allowance still admits a held registrable domain", attached: []string{"a.com", "b.com", "c.com"}, domain: "api.b.com", allowed: 1, accepted: true},
		{name: "overshot allowance refuses a new registrable domain", attached: []string{"a.com", "b.com", "c.com"}, domain: "d.com", allowed: 1, accepted: false},
		{name: "name without a registrable domain counts as its own unit", attached: []string{"co.uk"}, domain: "api.acme.com", allowed: 1, accepted: false},
		{name: "unlimited allowance admits the first domain", attached: nil, domain: "api.acme.com", allowed: domaingate.AllowanceUnlimited, accepted: true},
		{name: "unlimited allowance admits a new registrable domain", attached: []string{"a.com", "b.com", "c.com"}, domain: "d.com", allowed: domaingate.AllowanceUnlimited, accepted: true},
		{name: "unlimited allowance admits a name without a registrable domain", attached: []string{"co.uk"}, domain: "api.acme.com", allowed: domaingate.AllowanceUnlimited, accepted: true},
		{name: "allowance above unlimited admits a new registrable domain", attached: []string{"a.com"}, domain: "d.com", allowed: math.MaxUint32, accepted: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := domaingate.CheckAllowance(tc.attached, tc.domain, tc.allowed)
			if tc.accepted {
				require.NoError(t, err)
				return
			}
			requireCode(t, codes.Limits.CustomDomain.Exceeded, err)
		})
	}
}

func TestUnlimited(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		allowed   uint32
		unlimited bool
	}{
		{allowed: 0, unlimited: false},
		{allowed: 1, unlimited: false},
		{allowed: 999_999, unlimited: false},
		{allowed: 1_000_000, unlimited: true},
		{allowed: math.MaxUint32, unlimited: true},
	} {
		require.Equal(t, tc.unlimited, domaingate.Unlimited(tc.allowed), "allowance %d", tc.allowed)
	}
}

func TestAllowanceUsage(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		attached []string
		usage    int64
	}{
		{name: "no domains", attached: nil, usage: 0},
		{name: "one registrable domain", attached: []string{"api.acme.com", "app.acme.com", "acme.com", "v1.api.acme.com"}, usage: 1},
		{name: "two registrable domains", attached: []string{"api.acme.com", "app.acme.com", "api.other.com"}, usage: 2},
		{name: "multi-label suffix", attached: []string{"a.acme.co.uk", "b.acme.co.uk", "other.co.uk"}, usage: 2},
		{name: "private suffix", attached: []string{"a.github.io", "b.github.io"}, usage: 2},
		{name: "name without a registrable domain", attached: []string{"co.uk", "api.acme.co.uk"}, usage: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.usage, domaingate.AllowanceUsage(tc.attached))
		})
	}
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
		fault.UserFacingMessage(domaingate.CheckAllowance([]string{"api.acme.com"}, "api.other.com", 1)))

	require.Equal(t,
		"Resource limits are not configured for this workspace. Contact support@unkey.com.",
		fault.UserFacingMessage(domaingate.LimitsNotConfigured("ws_1234abcd")))

	// The allowance counts and the workspace id describe billing state the caller
	// cannot act on, so they stay internal.
	require.NotContains(t, fault.UserFacingMessage(domaingate.CheckAllowance([]string{"a.com", "b.com", "c.com"}, "d.com", 2)), "3")
	require.NotContains(t, fault.UserFacingMessage(domaingate.LimitsNotConfigured("ws_1234abcd")), "ws_1234abcd")
}
