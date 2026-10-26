package vault_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/pkg/uid"
	service "github.com/unkeyed/unkey/svc/vault"
	"github.com/unkeyed/unkey/svc/vault/internal/storage"
	"github.com/unkeyed/unkey/svc/vault/internal/vault"
	"github.com/unkeyed/unkey/svc/vault/keys"
)

type workloadClaims struct {
	jwt.RegisteredClaims
	WorkspaceID   string `json:"workspace_id"`
	AppID         string `json:"app_id"`
	EnvironmentID string `json:"environment_id"`
}

func TestOIDCNativeRPC(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	keyNext, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	kid := uid.New(uid.KeyPrefix)
	kidNext := uid.New(uid.KeyPrefix)
	var jwks atomic.Pointer[jose.JSONWebKeySet]
	jwks.Store(&jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: kid, Algorithm: "ES256", Use: "sig"}}})
	var discoveryCalls, jwksCalls atomic.Int64
	var issuerURL string
	issuer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var response any
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			discoveryCalls.Add(1)
			response = struct {
				Issuer  string `json:"issuer"`
				JWKSURI string `json:"jwks_uri"`
			}{Issuer: issuerURL, JWKSURI: issuerURL + "/keys"}
		case "/keys":
			jwksCalls.Add(1)
			response = jwks.Load()
		default:
			http.NotFound(w, r)
			return
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(issuer.Close)
	issuerURL = issuer.URL
	caPath := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: issuer.Certificate().Raw}), 0600))
	t.Setenv("SSL_CERT_FILE", caPath)
	t.Setenv("SSL_CERT_DIR", t.TempDir())
	subject := "spiffe://test.unkey/service/" + uid.New(uid.AppPrefix)
	_, masterKey, err := keys.GenerateMasterKey()
	require.NoError(t, err)
	cfg, err := config.LoadBytes[service.Config]([]byte(fmt.Sprintf(`
[oidc]
issuer = %q
audience = "vault"
allowed_subjects = [%q]
[encryption]
master_key = %q
[storage.disk]
path = %q
`, issuer.URL, subject, masterKey, t.TempDir())))
	require.NoError(t, err)
	store, err := storage.NewMemory()
	require.NoError(t, err)
	v, err := vault.New(vault.Config{Storage: store, MasterKey: cfg.Encryption.MasterKey, OIDC: cfg.OIDC})
	require.NoError(t, err)
	mux := http.NewServeMux()
	mux.Handle(vaultv1connect.NewVaultServiceHandler(v))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := vaultv1connect.NewVaultServiceClient(server.Client(), server.URL)
	claims := workloadClaims{
		RegisteredClaims: jwt.RegisteredClaims{Issuer: issuer.URL, Subject: subject, Audience: jwt.ClaimStrings{"vault"}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute))},
		WorkspaceID:      uid.New(uid.WorkspacePrefix), AppID: uid.New(uid.AppPrefix), EnvironmentID: uid.New(uid.EnvironmentPrefix),
	}
	sign := func(claims workloadClaims, key *ecdsa.PrivateKey, kid string) string {
		t.Helper()
		token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
		token.Header["kid"] = kid
		token.Header["jku"] = "https://attacker.invalid/keys"
		signed, err := token.SignedString(key)
		require.NoError(t, err)
		return signed
	}
	encrypt := func(token string, keyring string) (*connect.Response[vaultv1.EncryptResponse], error) {
		req := connect.NewRequest(&vaultv1.EncryptRequest{Keyring: keyring, Data: "secret"})
		if token != "" {
			req.Header().Set("Authorization", "Bearer "+token)
		}
		return client.Encrypt(t.Context(), req)
	}
	valid := sign(claims, key, kid)
	keyring := uid.New(uid.WorkspacePrefix)
	encrypted, err := encrypt(valid, keyring)
	require.NoError(t, err)
	decryptReq := connect.NewRequest(&vaultv1.DecryptRequest{Keyring: keyring, Encrypted: encrypted.Msg.GetEncrypted()})
	decryptReq.Header().Set("Authorization", "Bearer "+valid)
	decrypted, err := client.Decrypt(t.Context(), decryptReq)
	require.NoError(t, err)
	require.Equal(t, "secret", decrypted.Msg.GetPlaintext())

	for _, tc := range []struct {
		name   string
		mutate func(*workloadClaims)
	}{
		{"wrong subject with valid workspace", func(c *workloadClaims) { c.Subject += "/other" }},
		{"wrong audience", func(c *workloadClaims) { c.Audience = jwt.ClaimStrings{"other"} }},
		{"wrong issuer", func(c *workloadClaims) { c.Issuer += "/other" }},
		{"expired", func(c *workloadClaims) { c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute)) }},
		{"missing expiry", func(c *workloadClaims) { c.ExpiresAt = nil }},
		{"not yet valid", func(c *workloadClaims) { c.NotBefore = jwt.NewNumericDate(time.Now().Add(time.Hour)) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			invalid := claims
			tc.mutate(&invalid)
			_, err := encrypt(sign(invalid, key, kid), keyring)
			require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
		})
	}
	for _, token := range []string{"", "malformed", sign(claims, keyNext, kid), sign(claims, keyNext, kidNext)} {
		_, err := encrypt(token, keyring)
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	}
	_, err = client.Decrypt(t.Context(), connect.NewRequest(&vaultv1.DecryptRequest{}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = client.EncryptBulk(t.Context(), connect.NewRequest(&vaultv1.EncryptBulkRequest{}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = client.DecryptBulk(t.Context(), connect.NewRequest(&vaultv1.DecryptBulkRequest{}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = client.ReEncrypt(t.Context(), connect.NewRequest(&vaultv1.ReEncryptRequest{}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.EqualValues(t, 1, discoveryCalls.Load())
	require.EqualValues(t, 1, jwksCalls.Load())

	jwks.Store(&jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &keyNext.PublicKey, KeyID: kidNext, Algorithm: "ES256", Use: "sig"}}})
	rotated := sign(claims, keyNext, kidNext)
	require.Eventually(t, func() bool { _, err := encrypt(rotated, keyring); return err == nil }, 70*time.Second, time.Second)
	require.EqualValues(t, 2, jwksCalls.Load())
	_, err = encrypt(valid, keyring)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestAuthenticationConfig(t *testing.T) {
	for _, tc := range []struct {
		name, auth string
		valid      bool
	}{
		{"static", `bearer_token = "test"`, true},
		{"missing", "", false},
		{"oidc", "[oidc]\nissuer = 'https://issuer.example'\naudience = 'vault'\nallowed_subjects = ['spiffe://test/logdrain']", true},
		{"both", "bearer_token = 'test'\n[oidc]\nissuer = 'https://issuer.example'\naudience = 'vault'\nallowed_subjects = ['spiffe://test/logdrain']", false},
		{"http", "[oidc]\nissuer = 'http://issuer.example'\naudience = 'vault'\nallowed_subjects = ['spiffe://test/logdrain']", false},
		{"no subjects", "[oidc]\nissuer = 'https://issuer.example'\naudience = 'vault'", false},
		{"wildcard", "[oidc]\nissuer = 'https://issuer.example'\naudience = 'vault'\nallowed_subjects = ['spiffe://test/*']", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := config.LoadBytes[service.Config]([]byte(tc.auth + "\n[encryption]\nmaster_key = 'test'\n[storage.disk]\npath = '/tmp/vault'"))
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
