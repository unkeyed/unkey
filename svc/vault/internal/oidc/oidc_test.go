package oidc_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/vault/internal/oidc"
)

func TestDiscoveryFailsClosedAndBacksOff(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	for _, mode := range []string{"issuer mismatch", "http jwks", "oversized", "redirect", "timeout", "unavailable"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int64
			var issuerURL string
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				response := struct {
					Issuer  string `json:"issuer"`
					JWKSURI string `json:"jwks_uri"`
					Padding string `json:"padding,omitempty"`
				}{Issuer: issuerURL, JWKSURI: issuerURL + "/keys"}
				switch mode {
				case "issuer mismatch":
					response.Issuer += "/wrong"
				case "http jwks":
					response.JWKSURI = "http://attacker.invalid/keys"
				case "oversized":
					response.Padding = strings.Repeat("a", 1<<20)
				case "redirect":
					http.Redirect(w, r, issuerURL+"/redirect", http.StatusFound)
					return
				case "timeout":
					<-r.Context().Done()
					return
				case "unavailable":
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			issuerURL = server.URL
			caPath := filepath.Join(t.TempDir(), "ca.pem")
			require.NoError(t, os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0600))
			t.Setenv("SSL_CERT_FILE", caPath)
			t.Setenv("SSL_CERT_DIR", t.TempDir())
			subject := "spiffe://test/" + uid.New(uid.AppPrefix)
			v, err := oidc.New(oidc.Config{Issuer: issuerURL, Audience: "vault", AllowedSubjects: []string{subject}})
			require.NoError(t, err)
			token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.RegisteredClaims{Issuer: issuerURL, Subject: subject, Audience: jwt.ClaimStrings{"vault"}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute))})
			token.Header["kid"] = uid.New(uid.KeyPrefix)
			signed, err := token.SignedString(key)
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			started := time.Now()
			require.Error(t, v.Verify(ctx, signed))
			require.Less(t, time.Since(started), 7*time.Second)
			require.NoError(t, ctx.Err())
			for range 10 {
				require.Error(t, v.Verify(ctx, signed))
			}
			require.EqualValues(t, 1, calls.Load())
		})
	}
}
