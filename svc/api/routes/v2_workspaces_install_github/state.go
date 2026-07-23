package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/unkeyed/unkey/pkg/fault"
)

// This install-state signing is the Go counterpart of the dashboard's
// signState/verifyState in web/apps/dashboard/lib/trpc/routers/github.ts (see
// stateSigningKey, stableStringify, signState, verifyState). GitHub echoes the
// `state` back to the dashboard callback, which verifies it with that TS code,
// so the two MUST stay byte-compatible: same HMAC key derivation, the same
// canonical JSON (defined fields only, keys sorted, no HTML escaping), and the
// same unpadded base64url signature. Any change to the key derivation, the
// field set, or the serialization has to be made in BOTH files or verification
// breaks silently. The cross-language vectors in state_test.go guard this.

// installStateKeyPrefix domain-separates the HMAC key from the raw GitHub App
// private key. Must match the TS `unkey-github-install-state:` prefix.
const installStateKeyPrefix = "unkey-github-install-state:"

type signer struct {
	key []byte
}

// newSigner derives the HMAC key from the GitHub App private key PEM, matching
// the dashboard: SHA256(prefix + PEM). We normalize the PEM the same way the
// dashboard does before hashing so the derived key is identical on both sides:
//   - un-escape literal "\n" into real newlines (the dashboard's lib/env.ts),
//   - trim surrounding whitespace
func newSigner(privateKeyPEM string) *signer {
	normalized := strings.TrimSpace(strings.ReplaceAll(privateKeyPEM, `\n`, "\n"))
	sum := sha256.Sum256([]byte(installStateKeyPrefix + normalized))
	return &signer{key: sum[:]}
}

// payload is the claim set for the workspace-wide install flow. It carries no
// app, project, or repository; the dashboard signs its own app-targeted states.
// The JSON must match the dashboard's signer byte-for-byte, so empty optional
// fields are omitted rather than serialized.
type payload struct {
	WorkspaceID string
	Nonce       string
	ExpMs       int64
	// Source lets the dashboard discriminate the two flows. "api" marks a
	// workspace install, verified by the OAuth ownership proof rather than a
	// per-user binding.
	Source string
}

func (s *signer) sign(p payload) (string, error) {
	fields := p.fields()

	signed, err := canonicalJSON(fields)
	if err != nil {
		return "", err
	}

	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(signed))
	fields["sig"] = base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	// Transport encoding: key order is irrelevant since the dashboard re-parses
	// and re-canonicalizes (without `sig`) before recomputing the signature.
	out, err := canonicalJSON(fields)
	if err != nil {
		return "", err
	}
	return out, nil
}

func (p payload) fields() map[string]any {
	fields := map[string]any{
		"workspaceId": p.WorkspaceID,
		"nonce":       p.Nonce,
		"exp":         p.ExpMs,
	}
	if p.Source != "" {
		fields["source"] = p.Source
	}
	return fields
}

// canonicalJSON encodes the map the same way the dashboard's stableStringify
// does: keys sorted (encoding/json sorts map keys), no HTML escaping (TS
// JSON.stringify leaves <, >, & literal), and no trailing newline.
func canonicalJSON(fields map[string]any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(fields); err != nil {
		return "", fault.Wrap(err, fault.Internal("failed to encode github install state"))
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}
