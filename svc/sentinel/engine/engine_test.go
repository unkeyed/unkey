package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sentinelv1 "github.com/unkeyed/unkey/gen/proto/sentinel/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestParseMiddleware_Nil(t *testing.T) {
	t.Parallel()
	assert.Nil(t, ParseMiddleware(nil))
}

func TestParseMiddleware_Empty(t *testing.T) {
	t.Parallel()
	assert.Nil(t, ParseMiddleware([]byte{}))
}

func TestParseMiddleware_EmptyJSON(t *testing.T) {
	t.Parallel()
	assert.Nil(t, ParseMiddleware([]byte("{}")))
}

func TestParseMiddleware_InvalidProto(t *testing.T) {
	t.Parallel()
	assert.Nil(t, ParseMiddleware([]byte("not a valid protobuf")))
}

func TestParseMiddleware_NoPolicies(t *testing.T) {
	t.Parallel()
	//nolint:exhaustruct
	mw := &sentinelv1.Middleware{
		Policies: nil,
	}
	raw, err := protojson.Marshal(mw)
	require.NoError(t, err)
	assert.Nil(t, ParseMiddleware(raw))
}

func TestParseMiddleware_WithPolicies(t *testing.T) {
	t.Parallel()
	//nolint:exhaustruct
	mw := &sentinelv1.Middleware{
		Policies: []*sentinelv1.Policy{
			{
				Id:      "p1",
				Name:    "key auth",
				Enabled: true,
				Config: &sentinelv1.Policy_Keyauth{
					Keyauth: &sentinelv1.KeyAuth{KeySpaceId: "ks_123"},
				},
			},
		},
	}
	raw, err := protojson.Marshal(mw)
	require.NoError(t, err)

	result := ParseMiddleware(raw)
	require.NotNil(t, result)
	assert.Len(t, result.GetPolicies(), 1)
	assert.Equal(t, "p1", result.GetPolicies()[0].GetId())
}

func TestSerializePrincipal(t *testing.T) {
	t.Parallel()
	//nolint:exhaustruct
	p := &sentinelv1.Principal{
		Subject: "user_123",
		Type:    sentinelv1.PrincipalType_PRINCIPAL_TYPE_API_KEY,
		Claims: map[string]string{
			"key_id":       "key_abc",
			"workspace_id": "ws_456",
		},
	}

	s, err := SerializePrincipal(p)
	require.NoError(t, err)

	// Round-trip: unmarshal back into a Principal and compare
	var roundTripped sentinelv1.Principal
	err = protojson.Unmarshal([]byte(s), &roundTripped)
	require.NoError(t, err)
	assert.Equal(t, "user_123", roundTripped.GetSubject())
	assert.Equal(t, sentinelv1.PrincipalType_PRINCIPAL_TYPE_API_KEY, roundTripped.GetType())
	assert.Equal(t, "key_abc", roundTripped.GetClaims()["key_id"])
	assert.Equal(t, "ws_456", roundTripped.GetClaims()["workspace_id"])
}
