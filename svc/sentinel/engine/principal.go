package engine

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/ptr"
)

// Principal is the authenticated identity that sentinel produces after a
// successful authentication policy and forwards to the upstream on the
// X-Unkey-Principal header as JSON. The struct tags on this file are the
// authoritative wire contract — if any json tag, omitempty flag, or field
// type changes here, update docs/product/platform/sentinel/principal/*.mdx
// and the wire-format test in the same commit.
//
// The shape is decoupled from the sentinel proto schema on purpose:
//
//   - The Principal is produced by sentinel and consumed by user
//     applications via an HTTP header. It never crosses a proto wire, so
//     proto-as-IDL buys nothing.
//   - protojson has rigid behavior we have been fighting repeatedly: 64-bit
//     integers always serialize as JSON strings, enum values must be
//     SCREAMING_SNAKE_CASE, structpb dance for arbitrary JSON metadata.
//     encoding/json on a hand-written struct gives us full control over
//     the exact JSON we emit.
//
// Every Principal contains exactly one populated variant of Source matching
// Type. If multiple authentication policies could match, only the first
// successful one produces the Principal.
type Principal struct {
	// Version is the schema version of the Principal payload. Bumped on
	// breaking changes (removed/renamed fields, changed field types).
	// Additive changes do not bump it.
	//
	// Kept as a string so future schemes (date-based like "2026-04-13",
	// semver, named releases) can replace the current "v1" without
	// changing the field's type — which would itself be a breaking
	// change.
	Version string `json:"version"`

	// Subject is the primary identifier of the authenticated entity. For
	// API keys linked to an identity, it is the identity's external ID so
	// all keys under the same identity share a subject. For unlinked keys
	// it falls back to the key ID. For JWTs it is the configured subject
	// claim (default sub).
	Subject string `json:"subject"`

	// Type identifies which authentication method produced the Principal.
	// Always matches the populated variant of Source.
	Type PrincipalType `json:"type"`

	// Identity is the Unkey identity the credential is linked to, if any.
	// Omitted from the JSON entirely when no identity is linked.
	Identity *Identity `json:"identity,omitempty"`

	// Source carries the method-specific detail in a discriminated union.
	// Exactly one field inside Source is populated per request.
	Source Source `json:"source"`
}

// PrincipalType identifies the authentication method that produced a
// Principal. Kept as a named string so the JSON wire values are exactly
// "API_KEY" or "JWT" — no SCREAMING_SNAKE_CASE prefix, no enum number.
type PrincipalType string

const (
	// PrincipalTypeAPIKey is emitted when KeyAuth produced the Principal.
	PrincipalTypeAPIKey PrincipalType = "API_KEY"
	// PrincipalTypeJWT is emitted when JWTAuth produced the Principal.
	PrincipalTypeJWT PrincipalType = "JWT"
)

// PrincipalVersion is the current Principal schema version emitted by
// sentinel. Bumped on breaking changes to the JSON wire format.
const PrincipalVersion = "v1"

// Identity is the Unkey identity a credential is linked to. Grouping
// multiple credentials under a shared identity lets the upstream make
// decisions on the entity (a user, org, service account) rather than the
// individual credential.
type Identity struct {
	// ExternalID is the caller-assigned identifier for this identity.
	ExternalID string `json:"externalId"`

	// Meta is arbitrary metadata attached to the identity. Always emitted
	// as a JSON object — {} when no metadata was set. Callers must
	// initialize this to a non-nil map; a nil map would serialize to
	// null, which the contract forbids.
	Meta map[string]any `json:"meta"`
}

// Source is the discriminated union over method-specific detail. Exactly
// one field is populated per Principal; the populated field name matches
// the lowercase of Principal.Type (key for API_KEY, jwt for JWT).
type Source struct {
	// Key is populated when Type is API_KEY.
	Key *KeySource `json:"key,omitempty"`
	// JWT is populated when Type is JWT.
	JWT *JWTSource `json:"jwt,omitempty"`
}

// KeySource carries the verified API key detail.
type KeySource struct {
	// KeyID is the ID of the key that authenticated the request.
	KeyID string `json:"keyId"`

	// KeySpaceID is the keyspace the key belongs to.
	KeySpaceID string `json:"keySpaceId"`

	// Name is the human-readable name of the key, when set. Omitted
	// otherwise.
	Name *string `json:"name,omitempty"`

	// ExpiresAt is the Unix millisecond timestamp at which the key
	// expires. Omitted when the key has no expiry. Sentinel rejects
	// expired keys before producing a Principal, so if this field is
	// present the key is still valid.
	ExpiresAt *int64 `json:"expiresAt,omitempty"`

	// Meta is the custom metadata attached to the key. Always emitted
	// as a JSON object — {} when no metadata was set. Callers must
	// initialize this to a non-nil map.
	Meta map[string]any `json:"meta"`

	// Roles are the raw RBAC role names attached to the key. Omitted
	// from the JSON when no roles are attached.
	Roles []string `json:"roles,omitempty"`

	// Permissions are the raw RBAC permission strings attached to the
	// key. Omitted from the JSON when none are attached. Permission
	// queries configured on the KeyAuth policy are already enforced;
	// these are provided so the upstream can do further fine-grained
	// authorization on top.
	Permissions []string `json:"permissions,omitempty"`
}

// JWTSource carries the full decoded JWT. Claims are forwarded as-is,
// preserving the identity provider's names and types.
type JWTSource struct {
	// Header is the decoded token header (alg, typ, kid, ...).
	Header map[string]any `json:"header"`

	// Payload is the decoded token payload with all claims verbatim.
	Payload map[string]any `json:"payload"`

	// Signature is the raw signature string from the token's third
	// segment. Sentinel has already verified it; the upstream does not
	// need to re-verify.
	Signature string `json:"signature"`
}

// keyPrincipalFromVerifier builds an API_KEY Principal from a verified
// KeyVerifier. The caller is responsible for ensuring the key passed
// verification (StatusValid); this function treats every field on the
// verifier as trusted data and does no further validation.
//
// Returns an error only if key or identity metadata stored in the database
// is not valid JSON, which represents data corruption rather than user
// error.
func keyPrincipalFromVerifier(verifier *keys.KeyVerifier) (*Principal, error) {
	// When linked to an identity, use its external ID as the subject so
	// all keys under the same identity share one. Key.IdentityID and
	// Key.ExternalID are populated together by the identity join, so a
	// single gate on IdentityID covers both the subject upgrade and the
	// identity block.
	subject := verifier.Key.ID
	var identity *Identity
	if verifier.Key.IdentityID.Valid && verifier.Key.IdentityID.String != "" {
		subject = verifier.Key.ExternalID.String

		identityMeta, err := parseMetaBytes(verifier.Key.IdentityMeta)
		if err != nil {
			return nil, fmt.Errorf("parse identity meta: %w", err)
		}
		identity = &Identity{
			ExternalID: verifier.Key.ExternalID.String,
			Meta:       identityMeta,
		}
	}

	keyMeta, err := parseMetaString(verifier.Key.Meta)
	if err != nil {
		return nil, fmt.Errorf("parse key meta: %w", err)
	}

	var name *string
	if verifier.Key.Name.Valid && verifier.Key.Name.String != "" {
		name = ptr.P(verifier.Key.Name.String)
	}

	var expiresAt *int64
	if verifier.Key.Expires.Valid {
		expiresAt = ptr.P(verifier.Key.Expires.Time.UnixMilli())
	}

	return &Principal{
		Version:  PrincipalVersion,
		Subject:  subject,
		Type:     PrincipalTypeAPIKey,
		Identity: identity,
		Source: Source{
			Key: &KeySource{
				KeyID:       verifier.Key.ID,
				KeySpaceID:  verifier.Key.KeyAuthID,
				Name:        name,
				ExpiresAt:   expiresAt,
				Meta:        keyMeta,
				Roles:       verifier.Roles,
				Permissions: verifier.Permissions,
			},
			JWT: nil,
		},
	}, nil
}

// parseMetaString decodes a nullable JSON string column into a map. A null
// or empty column yields a non-nil empty map so the caller's field always
// serializes as `{}` rather than `null`.
func parseMetaString(meta sql.NullString) (map[string]any, error) {
	if !meta.Valid || meta.String == "" {
		return map[string]any{}, nil
	}
	return parseMetaBytes([]byte(meta.String))
}

// parseMetaBytes decodes raw JSON bytes into a map. Empty input yields a
// non-nil empty map. A JSON null at the root decodes to a nil map, which
// we normalize back to an empty map so downstream serialization is never
// `null`. Non-object JSON (an array or scalar at the root) returns an
// error — the contract is a JSON object.
func parseMetaBytes(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}
