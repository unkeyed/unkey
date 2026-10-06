package agentsignup

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth"
	authjwt "github.com/unkeyed/unkey/pkg/auth/jwt"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/auth/workos"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
)

const (
	localWorkOSSecret = "local-dev-jwt-secret-with-at-least-32-bytes"
	localAgentSecret  = "local-dev-agent-signup-jwt-secret-32b"
	localAgentIssuer  = "https://app.unkey.com/agent-signup"
	localAudience     = "api.unkey.com"
)

func TestLocalAuthChainAcceptsDashboardHS256Token(t *testing.T) {
	t.Parallel()

	lookup := authjwt.WorkspaceLookupFunc(func(_ context.Context, orgID string) (string, error) {
		require.Equal(t, "org_123", orgID)
		return "ws_123", nil
	})
	workosResolver, err := authjwt.NewResolver(lookup, "app.unkey.com", localAudience, []byte(localWorkOSSecret))
	require.NoError(t, err)
	agentResolver, err := authjwt.NewResolver(lookup, localAgentIssuer, localAudience, []byte(localAgentSecret))
	require.NoError(t, err)
	service := auth.New(
		workos.NewRoleMappingResolver(workosResolver),
		NewRoleMappingResolver(agentResolver),
	)

	now := time.Now().Unix()
	token := signHS256(t, []byte(localAgentSecret), map[string]any{
		"alg": "HS256",
		"typ": "JWT",
	}, map[string]any{
		"org":   map[string]any{"id": "org_123"},
		"name":  "user_abc",
		"roles": []string{Role},
		"iss":   localAgentIssuer,
		"aud":   []string{localAudience},
		"sub":   "user_abc",
		"iat":   now,
		"nbf":   now,
		"exp":   now + 120,
	})

	req := httptest.NewRequest(http.MethodPost, "/v2/rootKeys.createAgentKey", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))

	resolved, err := service.Authenticate(context.Background(), sess)
	require.NoError(t, err)
	require.Equal(t, principal.TypeJWT, resolved.Type)
	require.Equal(t, "user_abc", resolved.Subject.ID)
	require.Equal(t, "ws_123", resolved.AuthorizedWorkspaceID)
	source, ok := resolved.Source.(principal.JWTSource)
	require.True(t, ok)
	require.Equal(t, []string{Role}, source.Roles)
	require.Equal(t, "HS256", source.Header["alg"])
	_, hasKID := source.Header["kid"]
	require.False(t, hasKID)
	require.Contains(t, resolved.Permissions, "unkey:v1:ws_123:projects/*/keyspaces/*#read")
}

func TestLocalAuthChainRejectsJoseStringAudience(t *testing.T) {
	t.Parallel()

	lookup := authjwt.WorkspaceLookupFunc(func(context.Context, string) (string, error) {
		return "ws_123", nil
	})
	workosResolver, err := authjwt.NewResolver(lookup, "app.unkey.com", localAudience, []byte(localWorkOSSecret))
	require.NoError(t, err)
	agentResolver, err := authjwt.NewResolver(lookup, localAgentIssuer, localAudience, []byte(localAgentSecret))
	require.NoError(t, err)
	service := auth.New(
		workos.NewRoleMappingResolver(workosResolver),
		NewRoleMappingResolver(agentResolver),
	)

	now := time.Now().Unix()
	token := signHS256(t, []byte(localAgentSecret), map[string]any{
		"alg": "HS256",
		"typ": "JWT",
	}, map[string]any{
		"org":   map[string]any{"id": "org_123"},
		"name":  "user_abc",
		"roles": []string{Role},
		"iss":   localAgentIssuer,
		"aud":   localAudience,
		"sub":   "user_abc",
		"iat":   now,
		"nbf":   now,
		"exp":   now + 120,
	})

	req := httptest.NewRequest(http.MethodPost, "/v2/rootKeys.createAgentKey", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))

	_, err = service.Authenticate(context.Background(), sess)
	require.Error(t, err)
	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Auth.Authentication.Malformed.URN(), code)
}

func signHS256(t *testing.T, secret []byte, header map[string]any, payload map[string]any) string {
	t.Helper()
	headerJSON, err := json.Marshal(header)
	require.NoError(t, err)
	payloadJSON, err := json.Marshal(payload)
	require.NoError(t, err)
	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	mac := hmac.New(sha256.New, secret)
	_, err = mac.Write([]byte(encodedHeader + "." + encodedPayload))
	require.NoError(t, err)
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return encodedHeader + "." + encodedPayload + "." + signature
}
