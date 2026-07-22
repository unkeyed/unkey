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

// signer holds only the derived HMAC key, never the raw private key.
type signer struct {
	key []byte
}

// newSigner derives the HMAC key from the GitHub App private key PEM, matching
// the dashboard: SHA256(prefix + PEM). The dashboard un-escapes literal "\n"
// into real newlines before hashing (lib/env.ts), so we normalize the same way;
// a PEM that already carries real newlines is unaffected.
func newSigner(privateKeyPEM string) *signer {
	normalized := strings.ReplaceAll(privateKeyPEM, `\n`, "\n")
	sum := sha256.Sum256([]byte(installStateKeyPrefix + normalized))
	return &signer{key: sum[:]}
}

// payload is the install-state claim set. Optional fields are omitted from the
// signed JSON when empty, matching how the dashboard drops undefined properties.
type payload struct {
	ProjectID   string
	AppID       string
	WorkspaceID string
	Nonce       string
	ExpMs       int64
	// ReturnTo routes the dashboard callback ("settings" or empty for the
	// default select-repo step).
	ReturnTo string
	// Repository ("owner/name"), when set, tells the callback to auto-connect
	// the repo instead of showing the picker.
	Repository string
	// Source marks the flow origin. "api" signals the callback to skip the
	// dashboard-user binding while keeping the workspace binding and the OAuth
	// ownership proof.
	Source string
}

// sign returns the JSON state string to place in the GitHub install URL's
// `state` query parameter.
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

// fields builds the signed claim map with only the defined fields present.
func (p payload) fields() map[string]any {
	fields := map[string]any{
		"projectId":   p.ProjectID,
		"appId":       p.AppID,
		"workspaceId": p.WorkspaceID,
		"nonce":       p.Nonce,
		"exp":         p.ExpMs,
	}
	if p.ReturnTo != "" {
		fields["returnTo"] = p.ReturnTo
	}
	if p.Repository != "" {
		fields["repository"] = p.Repository
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
