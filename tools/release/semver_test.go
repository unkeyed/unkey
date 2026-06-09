package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func mustParse(t *testing.T, s string) semver {
	t.Helper()
	v, ok := parseSemver(s)
	require.True(t, ok, "parse %q", s)
	return v
}

func TestParseAndFormatRoundTrip(t *testing.T) {
	for _, s := range []string{"v1.2.3", "v0.0.1", "v10.20.30", "v1.2.3-rc.1", "v2.0.0-alpha.3", "v0.4.0-beta"} {
		require.Equal(t, s, formatVersion(mustParse(t, s)))
	}
}

func TestParseRejectsInvalid(t *testing.T) {
	for _, s := range []string{"1.2.3", "v1.2", "vx.y.z", "v1.2.3.4", ""} {
		_, ok := parseSemver(s)
		require.False(t, ok, "expected %q to be invalid", s)
	}
}

func TestValidSemver(t *testing.T) {
	valid := []string{
		"v1.2.3", "v0.0.1", "v10.20.30",
		"v1.2.3-rc.1", "v0.4.0-beta", "v2.0.0-alpha.3",
		"v1.0.0-alpha-beta", // hyphen inside an identifier is legal (SemVer §9)
		"v1.0.0-rc--1",      // consecutive hyphens form one valid identifier
	}
	for _, s := range valid {
		require.NoError(t, validSemver(s), "expected %q to be valid", s)
	}

	invalid := []string{
		"1.2.3", "v1.2", "v1.2.3.4", "",
		"v1.2.3-rc..1", // empty identifier from consecutive dots
		"v1.2.3-rc.",   // trailing dot leaves an empty identifier
		"v1.2.3-.rc",   // leading dot leaves an empty identifier
	}
	for _, s := range invalid {
		require.Error(t, validSemver(s), "expected %q to be invalid", s)
	}
}

func TestCmpPrecedence(t *testing.T) {
	require.Equal(t, 1, cmp(mustParse(t, "v1.2.4"), mustParse(t, "v1.2.3")))
	require.Equal(t, 1, cmp(mustParse(t, "v1.3.0"), mustParse(t, "v1.2.9")))
	// Stable outranks a pre-release of the same base.
	require.Equal(t, 1, cmp(mustParse(t, "v1.2.3"), mustParse(t, "v1.2.3-rc.1")))
	// Higher rc number is greater.
	require.Equal(t, 1, cmp(mustParse(t, "v1.2.3-rc.2"), mustParse(t, "v1.2.3-rc.1")))
	// alpha < beta < rc lexically.
	require.Equal(t, -1, cmp(mustParse(t, "v1.0.0-alpha"), mustParse(t, "v1.0.0-beta")))
}

func TestMaxStableIgnoresPreReleases(t *testing.T) {
	versions := []semver{
		mustParse(t, "v1.2.3"),
		mustParse(t, "v1.3.0-rc.5"),
		mustParse(t, "v1.2.2"),
	}
	latest, ok := maxStable(versions)
	require.True(t, ok)
	require.Equal(t, "v1.2.3", formatVersion(latest))
}

func TestMaxStableNoneFound(t *testing.T) {
	_, ok := maxStable([]semver{mustParse(t, "v1.0.0-rc.1")})
	require.False(t, ok)
}

func TestBumpVersion(t *testing.T) {
	base := mustParse(t, "v1.2.3")
	require.Equal(t, "v1.2.4", formatVersion(bumpVersion(base, "patch")))
	require.Equal(t, "v1.3.0", formatVersion(bumpVersion(base, "minor")))
	require.Equal(t, "v2.0.0", formatVersion(bumpVersion(base, "major")))
	// Default kind is patch.
	require.Equal(t, "v1.2.4", formatVersion(bumpVersion(base, "")))
	// Pre-release identifiers are cleared.
	require.Equal(t, "v1.2.4", formatVersion(bumpVersion(mustParse(t, "v1.2.3-rc.1"), "patch")))
}

func TestNextPreNumber(t *testing.T) {
	target := mustParse(t, "v1.2.4")
	// No existing rc -> 1.
	require.Equal(t, 1, nextPreNumber(nil, target, "rc"))
	// Existing rc.2 -> 3.
	versions := []semver{mustParse(t, "v1.2.4-rc.1"), mustParse(t, "v1.2.4-rc.2")}
	require.Equal(t, 3, nextPreNumber(versions, target, "rc"))
	// rc tags for a different base are ignored.
	other := []semver{mustParse(t, "v1.3.0-rc.5")}
	require.Equal(t, 1, nextPreNumber(other, target, "rc"))
	// Different label is ignored.
	require.Equal(t, 1, nextPreNumber([]semver{mustParse(t, "v1.2.4-beta.4")}, target, "rc"))
	// Bare label with no number counts as 0 -> next 1.
	require.Equal(t, 1, nextPreNumber([]semver{mustParse(t, "v1.2.4-rc")}, target, "rc"))
}
