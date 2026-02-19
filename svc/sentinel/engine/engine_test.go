package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
	sentinelv1 "github.com/unkeyed/unkey/gen/proto/sentinel/v1"
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
	//nolint:exhaustruct
	mw := &sentinelv1.Config{
		Policies: nil,
	}
	raw, err := protojson.Marshal(mw)
	require.NoError(t, err)

	policies, err := ParseMiddleware(raw)
	require.Nil(t, err)
	require.Nil(t, policies)
}

func TestParseMiddleware_WithPolicies(t *testing.T) {
	t.Parallel()
	//nolint:exhaustruct
	mw := &sentinelv1.Config{
		Policies: []*sentinelv1.Policy{
			{
				Id:      "p1",
				Name:    "key auth",
				Enabled: true,
				Config: &sentinelv1.Policy_Keyauth{
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
	require.Equal(t, "user_123", roundTripped.GetSubject())
	require.Equal(t, sentinelv1.PrincipalType_PRINCIPAL_TYPE_API_KEY, roundTripped.GetType())
	require.Equal(t, "key_abc", roundTripped.GetClaims()["key_id"])
	require.Equal(t, "ws_456", roundTripped.GetClaims()["workspace_id"])
}
