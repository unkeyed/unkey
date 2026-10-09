package jwt

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth"
	"github.com/unkeyed/unkey/pkg/auth/workos"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestWorkOSChain_AudienceFallthroughAndCeiling guarantees two WorkOS JWKS
// entries accept only their own audience, a mismatch falls through to the
// next entry, and the matching entry's permission ceiling is what the
// principal receives. The token aud is a JSON string, matching AuthKit.
func TestWorkOSChain_AudienceFallthroughAndCeiling(t *testing.T) {
	t.Parallel()

	privateKeyPEM, jwksBody := generateJWKS(t)
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(jwksBody)
		require.NoError(t, err)
	}))
	t.Cleanup(jwksServer.Close)

	const (
		issuer          = "https://beautiful-day-63-staging.authkit.app"
		audienceV2      = "https://mcp.unkey.com/mcp/v2"
		audienceCompute = "https://mcp.unkey.com/mcp/compute"
	)
	orgID := uid.New(uid.OrgPrefix)
	workspaceID := uid.New(uid.WorkspacePrefix)
	subject := uid.New(uid.TestPrefix)
	v2Ceiling := []string{"projects/*#read"}
	computeCeiling := []string{"projects/*/keyspaces/*#read"}

	entry := func(audience string, ceiling []string) auth.Resolver {
		t.Helper()
		resolver, err := NewResolverWithJWKSURL(testWorkspaceLookup(workspaceID), issuer, audience, jwksServer.URL)
		require.NoError(t, err)
		return workos.NewRoleMappingResolver(resolver, ceiling)
	}
	v2First := auth.New(entry(audienceV2, v2Ceiling), entry(audienceCompute, computeCeiling))
	computeFirst := auth.New(entry(audienceCompute, computeCeiling), entry(audienceV2, v2Ceiling))
	computeOnly := auth.New(entry(audienceCompute, computeCeiling))

	now := time.Now().Unix()
	adminV2 := signConnectToken(t, privateKeyPEM, connectToken{
		Issuer:    issuer,
		Audience:  audienceV2,
		Subject:   subject,
		OrgID:     orgID,
		ExpiresAt: now + 300,
		IssuedAt:  now,
		Roles:     []string{"admin"},
	})

	principal, err := v2First.Authenticate(context.Background(), jwtSession(t, adminV2))
	require.NoError(t, err)
	require.Equal(t, subject, principal.Subject.ID)
	require.Equal(t, workspaceID, principal.AuthorizedWorkspaceID)
	require.Equal(t, []string{"unkey:v1:" + workspaceID + ":projects/*#read"}, principal.Permissions)

	principal, err = computeFirst.Authenticate(context.Background(), jwtSession(t, adminV2))
	require.NoError(t, err)
	require.Equal(t, []string{"unkey:v1:" + workspaceID + ":projects/*#read"}, principal.Permissions)

	principal, err = computeOnly.Authenticate(context.Background(), jwtSession(t, adminV2))
	require.Error(t, err)
	require.Nil(t, principal)

	expired := signConnectToken(t, privateKeyPEM, connectToken{
		Issuer:    issuer,
		Audience:  audienceV2,
		Subject:   subject,
		OrgID:     orgID,
		ExpiresAt: now - 60,
		IssuedAt:  now - 120,
		Roles:     []string{"admin"},
	})
	principal, err = v2First.Authenticate(context.Background(), jwtSession(t, expired))
	require.Error(t, err)
	require.Nil(t, principal)

	wrongIssuer := signConnectToken(t, privateKeyPEM, connectToken{
		Issuer:    "https://login.unkey.com",
		Audience:  audienceV2,
		Subject:   subject,
		OrgID:     orgID,
		ExpiresAt: now + 300,
		IssuedAt:  now,
		Roles:     []string{"admin"},
	})
	principal, err = v2First.Authenticate(context.Background(), jwtSession(t, wrongIssuer))
	require.Error(t, err)
	require.Nil(t, principal)

	noRoles := signConnectToken(t, privateKeyPEM, connectToken{
		Issuer:    issuer,
		Audience:  audienceV2,
		Subject:   subject,
		OrgID:     orgID,
		ExpiresAt: now + 300,
		IssuedAt:  now,
	})
	principal, err = v2First.Authenticate(context.Background(), jwtSession(t, noRoles))
	require.NoError(t, err)
	require.Equal(t, subject, principal.Subject.ID)
	require.Empty(t, principal.Permissions)
}

type connectToken struct {
	Issuer    string
	Audience  string
	Subject   string
	OrgID     string
	ExpiresAt int64
	IssuedAt  int64
	Roles     []string
}

func signConnectToken(t *testing.T, privateKeyPEM string, token connectToken) string {
	t.Helper()

	block, _ := pem.Decode([]byte(privateKeyPEM))
	require.NotNil(t, block)
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(t, err)

	header, err := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT", "kid": "test-key-1"})
	require.NoError(t, err)
	payload := map[string]any{
		"iss":       token.Issuer,
		"aud":       token.Audience,
		"sub":       token.Subject,
		"sid":       uid.New(uid.TestPrefix),
		"client_id": uid.New(uid.TestPrefix),
		"scope":     "openid profile email",
		"org_id":    token.OrgID,
		"org":       map[string]string{"id": token.OrgID},
		"exp":       token.ExpiresAt,
		"iat":       token.IssuedAt,
	}
	if token.Roles != nil {
		payload["roles"] = token.Roles
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	signingInput := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(body)
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	require.NoError(t, err)
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
}
