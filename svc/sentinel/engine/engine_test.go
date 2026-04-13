package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
	sentinelv1 "github.com/unkeyed/unkey/gen/proto/sentinel/v1"
	"github.com/unkeyed/unkey/pkg/ptr"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestParseMiddleware_Nil(t *testing.T) {
	t.Parallel()
	policies, err := ParseMiddleware(nil)
	require.Nil(t, err)
	require.Nil(t, policies)
}

func TestParseMiddleware_Empty(t *testing.T) {
	t.Parallel()
	policies, err := ParseMiddleware([]byte{})
	require.Nil(t, err)
	require.Nil(t, policies)
}

func TestParseMiddleware_EmptyJSON(t *testing.T) {
	t.Parallel()
	policies, err := ParseMiddleware([]byte("{}"))
	require.Nil(t, err)
	require.Nil(t, policies)
}

func TestParseMiddleware_InvalidProto(t *testing.T) {
	t.Parallel()
	policies, err := ParseMiddleware([]byte("not a valid protobuf"))
	require.Error(t, err)
	require.Nil(t, policies)
}

func TestParseMiddleware_NoPolicies(t *testing.T) {
	t.Parallel()
	mw := &sentinelv1.Config{Policies: nil}
	raw, err := protojson.Marshal(mw)
	require.NoError(t, err)

	policies, err := ParseMiddleware(raw)
	require.Nil(t, err)
	require.Nil(t, policies)
}

func TestParseMiddleware_WithPolicies(t *testing.T) {
	t.Parallel()
	mw := &sentinelv1.Config{
		Policies: []*sentinelv1.Policy{
			{
				Id:      "p1",
				Name:    "key auth",
				Enabled: true,
				Match:   nil,
				Config: &sentinelv1.Policy_Keyauth{
					//nolint:exhaustruct
					Keyauth: &sentinelv1.KeyAuth{KeySpaceIds: []string{"ks_123"}},
				},
			},
		},
	}
	raw, err := protojson.Marshal(mw)
	require.NoError(t, err)

	policies, err := ParseMiddleware(raw)
	require.NoError(t, err)
	require.NotNil(t, policies)
	require.Len(t, policies, 1)
	require.Equal(t, "p1", policies[0].GetId())
}

// TestPrincipal_Marshal_WireFormat pins the JSON wire format of the
// Principal. The header contract documented in
// docs/product/platform/sentinel/principal/overview.mdx is exactly this
// output — if this test changes, update the docs in the same commit.
func TestPrincipal_Marshal_WireFormat(t *testing.T) {
	t.Parallel()

	t.Run("minimal principal omits optional fields", func(t *testing.T) {
		t.Parallel()
		p := &Principal{
			Version: PrincipalVersion,
			Subject: "key_abc",
			Type:    PrincipalTypeAPIKey,
			Source: Source{
				Key: &KeySource{
					KeyID:       "key_abc",
					KeySpaceID:  "ks_456",
					Name:        nil,
					ExpiresAt:   nil,
					Meta:        map[string]any{},
					Roles:       nil,
					Permissions: nil,
				},
				JWT: nil,
			},
		}

		s, err := p.Marshal()
		require.NoError(t, err)
		require.JSONEq(t, `{
			"version": "v1",
			"subject": "key_abc",
			"type": "API_KEY",
			"source": {"key": {
				"keyId": "key_abc",
				"keySpaceId": "ks_456",
				"meta": {}
			}}
		}`, s)
	})

	t.Run("populated principal includes identity and optional key fields", func(t *testing.T) {
		t.Parallel()
		p := &Principal{
			Version: PrincipalVersion,
			Subject: "user_42",
			Type:    PrincipalTypeAPIKey,
			Identity: &Identity{
				ExternalID: "user_42",
				Meta:       map[string]any{"plan": "pro"},
			},
			Source: Source{
				Key: &KeySource{
					KeyID:       "key_abc",
					KeySpaceID:  "ks_456",
					Name:        ptr.P("prod"),
					ExpiresAt:   ptr.P(int64(1717200000000)),
					Meta:        map[string]any{},
					Roles:       []string{"admin"},
					Permissions: []string{"api.read", "api.write"},
				},
				JWT: nil,
			},
		}

		s, err := p.Marshal()
		require.NoError(t, err)
		require.JSONEq(t, `{
			"version": "v1",
			"subject": "user_42",
			"type": "API_KEY",
			"identity": {"externalId": "user_42", "meta": {"plan": "pro"}},
			"source": {"key": {
				"keyId": "key_abc",
				"keySpaceId": "ks_456",
				"name": "prod",
				"expiresAt": 1717200000000,
				"meta": {},
				"roles": ["admin"],
				"permissions": ["api.read", "api.write"]
			}}
		}`, s)
	})
}
