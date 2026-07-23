package handler

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// The expected signatures are computed with the dashboard's exact algorithm
// (Node crypto): key = SHA256("unkey-github-install-state:"+PEM),
// sig = base64url(HMAC-SHA256(key, JSON.stringify(payload, sortedKeys))).
// If either implementation drifts, these vectors fail. Regenerate with the
// snippet in the PR description if the scheme intentionally changes.
const testPEM = "test-pem-value"

func sigFromState(t *testing.T, state string) (string, map[string]any) {
	t.Helper()
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(state), &parsed))
	sig, ok := parsed["sig"].(string)
	require.True(t, ok, "state must carry a string sig")
	return sig, parsed
}

// This endpoint mints only the workspace install-only flow: workspaceId, nonce,
// exp, and source, with no projectId/appId/repository/returnTo. The vector pins
// that shape against the dashboard's verifyState.
func TestSign_MatchesDashboardVector_InstallOnly(t *testing.T) {
	s := newSigner(testPEM)

	state, err := s.sign(payload{
		WorkspaceID: "ws_KEBAP",
		Nonce:       "nonce123",
		ExpMs:       1700000000000,
		Source:      "api",
	})
	require.NoError(t, err)

	sig, parsed := sigFromState(t, state)
	require.Equal(t, "_YHr1IIkmnCzxPti5-GosO-PXHW4xMPoI06AVdyCNl0", sig)
	require.Equal(t, "api", parsed["source"])
	require.NotContains(t, parsed, "projectId")
	require.NotContains(t, parsed, "appId")
	require.NotContains(t, parsed, "repository")
	require.NotContains(t, parsed, "returnTo")
}

func sigOf(t *testing.T, s *signer, p payload) string {
	t.Helper()
	state, err := s.sign(p)
	require.NoError(t, err)
	sig, _ := sigFromState(t, state)
	return sig
}

// Every field must be covered by the signature: flipping any one of them must
// change the sig, otherwise an attacker who tampers with that field in the
// callback would not be detected by the dashboard's verifyState.
func TestSign_EveryFieldIsAuthenticated(t *testing.T) {
	s := newSigner(testPEM)
	base := payload{
		WorkspaceID: "ws_1",
		Nonce:       "nonce_1",
		ExpMs:       1700000000000,
	}
	baseSig := sigOf(t, s, base)

	mutate := func(f func(*payload)) payload {
		p := base
		f(&p)
		return p
	}

	for _, tc := range []struct {
		name string
		p    payload
	}{
		{"workspaceId", mutate(func(p *payload) { p.WorkspaceID = "ws_2" })},
		{"nonce", mutate(func(p *payload) { p.Nonce = "nonce_2" })},
		{"exp", mutate(func(p *payload) { p.ExpMs = base.ExpMs + 1 })},
		{"source", mutate(func(p *payload) { p.Source = "api" })},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NotEqual(t, baseSig, sigOf(t, s, tc.p), "changing %s must change the signature", tc.name)
		})
	}
}

// The signature is bound to the signing key: a different private key produces a
// different sig, so a state cannot be forged without the server secret.
func TestSign_KeyBound(t *testing.T) {
	base := payload{WorkspaceID: "w", Nonce: "n", ExpMs: 1}
	require.NotEqual(t,
		sigOf(t, newSigner(testPEM), base),
		sigOf(t, newSigner("a-different-private-key"), base),
	)
}

func TestSign_Deterministic(t *testing.T) {
	s := newSigner(testPEM)
	base := payload{WorkspaceID: "w", Nonce: "n", ExpMs: 1}
	require.Equal(t, sigOf(t, s, base), sigOf(t, s, base))
}

// PEM normalization: a private key stored with literal "\n" escapes must derive
// the same key as one with real newlines, matching the dashboard's un-escaping.
func TestNewSigner_NormalizesEscapedNewlines(t *testing.T) {
	escaped := newSigner(`line1\nline2`)
	real := newSigner("line1\nline2")

	p := payload{WorkspaceID: "w", Nonce: "n", ExpMs: 1}
	got1, err := escaped.sign(p)
	require.NoError(t, err)
	got2, err := real.sign(p)
	require.NoError(t, err)
	require.Equal(t, got2, got1)
}
