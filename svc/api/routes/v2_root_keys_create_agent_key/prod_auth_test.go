package handler_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth"
	"github.com/unkeyed/unkey/pkg/auth/agentsignup"
	authjwt "github.com/unkeyed/unkey/pkg/auth/jwt"
	"github.com/unkeyed/unkey/pkg/db"
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/svc/api/internal/middleware"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_root_keys_create_agent_key"
)

func TestCreateAgentKeyWithProductionJWKSAuth(t *testing.T) {
	privateKeyPEM, jwksBody := agentSignupJWKS(t)
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(jwksBody)
		require.NoError(t, err)
	}))
	t.Cleanup(jwks.Close)

	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace

	jwtResolver, err := authjwt.NewResolverWithJWKSURL(
		authjwt.WorkspaceLookupFunc(func(_ context.Context, orgID string) (string, error) {
			if orgID != workspace.OrgID {
				return "", authjwt.ErrWorkspaceNotFound
			}
			return workspace.ID, nil
		}),
		agentsignup.Issuer,
		authjwt.Audience,
		jwks.URL,
	)
	require.NoError(t, err)
	resolver := agentsignup.NewRoleMappingResolver(jwtResolver)

	route := &handler.Handler{
		DB: h.DB, Keys: h.Keys, Auditlogs: h.Auditlogs, Clock: h.Clock,
	}
	h.Register(route, append(h.PublicMiddleware(), middleware.WithAuthentication(middleware.AuthenticationConfig{
		Auth:             auth.New(resolver),
		KeyVerifications: h.KeyVerifications,
		Region:           "test",
		Database:         h.DB,
		LimitsCache:      h.Caches.WorkspaceLimits,
		Ratelimit:        h.Ratelimit,
	}))...)

	signer, err := tokenjwt.NewRS256Signer[authjwt.Claims](privateKeyPEM)
	require.NoError(t, err)
	now := time.Now()
	token, err := signer.Sign(authjwt.Claims{
		RegisteredClaims: tokenjwt.RegisteredClaims{
			Issuer:    agentsignup.Issuer,
			Subject:   "user_agent",
			Audience:  []string{authjwt.Audience},
			ExpiresAt: now.Add(time.Minute).Unix(),
			NotBefore: now.Add(-time.Second).Unix(),
			IssuedAt:  now.Unix(),
		},
		Org:   authjwt.OrganizationClaims{ID: workspace.OrgID},
		Roles: []string{agentsignup.Role},
	})
	require.NoError(t, err)

	headers := http.Header{
		"Authorization": {"Bearer " + token},
		"Content-Type":  {"application/json"},
	}
	allowed := "unkey:v1:" + workspace.ID + ":projects/*/keyspaces/*#read"
	for _, permission := range []string{
		"unkey:v1:" + workspace.ID + ":projects/*/keyspaces/*/keys/*#decrypt",
		"unkey:v1:" + workspace.ID + ":**#*",
		"unkey:v1:" + workspace.ID + ":rootKeys/*#write",
	} {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Permissions: []string{permission},
		})
		require.Equal(t, http.StatusForbidden, res.Status, "%s", res.RawBody)
	}

	before, err := db.Query.ListRootKeys(t.Context(), h.DB.RO(), db.ListRootKeysParams{
		WorkspaceID: workspace.ID,
		IDCursor:    "",
		Limit:       100,
	})
	require.NoError(t, err)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Name:        nil,
		Permissions: []string{allowed},
	})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	require.NotEmpty(t, res.Body.Data.Key)
	require.NotEmpty(t, res.Body.Data.KeyId)

	stored, err := db.Query.FindUnkeyRootKeyByID(t.Context(), h.DB.RO(), res.Body.Data.KeyId)
	require.NoError(t, err)
	require.NotEmpty(t, stored.Hash)
	require.NotEqual(t, res.Body.Data.Key, stored.Hash)
	grants, err := db.Query.ListUnkeyPermissionsByPrincipal(t.Context(), h.DB.RO(), db.ListUnkeyPermissionsByPrincipalParams{
		WorkspaceID:   workspace.ID,
		PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeRootKey,
		PrincipalID:   res.Body.Data.KeyId,
	})
	require.NoError(t, err)
	require.Equal(t, []string{allowed}, grants)

	after, err := db.Query.ListRootKeys(t.Context(), h.DB.RO(), db.ListRootKeysParams{
		WorkspaceID: workspace.ID,
		IDCursor:    "",
		Limit:       100,
	})
	require.NoError(t, err)
	require.Len(t, after, len(before)+1)
}

func agentSignupJWKS(t *testing.T) (string, []byte) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privateKeyPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}))
	exponent := big.NewInt(int64(privateKey.PublicKey.E)).Bytes()
	body, err := json.Marshal(map[string]any{
		"keys": []map[string]string{
			{
				"alg": "RS256",
				"kty": "RSA",
				"use": "sig",
				"kid": "agent-signup",
				"n":   base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(exponent),
			},
		},
	})
	require.NoError(t, err)
	return privateKeyPEM, body
}
