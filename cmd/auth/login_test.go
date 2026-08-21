package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/internal/oauth"
	"github.com/unkeyed/unkey/pkg/cli"
)

func jwtWithExp(t *testing.T, exp time.Time) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256"}`))
	payload, err := json.Marshal(map[string]any{"exp": exp.Unix()})
	require.NoError(t, err)
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}

// deviceFlowServer returns a server that completes the device flow, minting a
// token with the given organization_id and email.
func deviceFlowServer(t *testing.T, orgID, email string) *httptest.Server {
	t.Helper()
	exp := time.Now().Add(time.Hour)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/user_management/authorize/device":
			fmt.Fprint(w, `{"device_code":"dc","user_code":"WDJB-MJHT","verification_uri":"https://work.os/device","verification_uri_complete":"https://work.os/device?code=WDJB-MJHT","expires_in":300,"interval":1}`)
		case "/user_management/authenticate":
			b, _ := json.Marshal(map[string]any{
				"access_token":    jwtWithExp(t, exp),
				"refresh_token":   "refresh-value",
				"organization_id": orgID,
				"user":            map[string]any{"email": email},
			})
			fmt.Fprint(w, string(b))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
}

func TestDeviceLogin_Success(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Pre-existing root key must survive an OAuth login.
	require.NoError(t, cli.SaveUserConfig(cli.UserConfig{RootKey: "existing_root"}))

	srv := deviceFlowServer(t, "org_42", "james@unkey.com")
	defer srv.Close()

	var out bytes.Buffer
	client := oauth.New(oauth.WithBaseURL(srv.URL))
	err := deviceLogin(context.Background(), client, "client_abc", &out, nil)
	require.NoError(t, err)

	path, _ := cli.UserConfigPath()
	cfg, err := cli.LoadUserConfig(path)
	require.NoError(t, err)
	require.Equal(t, "org_42", cfg.OrgID)
	require.Equal(t, "james@unkey.com", cfg.UserEmail)
	require.Equal(t, cli.Secret("refresh-value"), cfg.RefreshToken)
	require.True(t, cfg.HasOAuth())
	require.Equal(t, "existing_root", cfg.RootKey, "root key should be preserved")
	require.Contains(t, out.String(), "Logged in as james@unkey.com (org: org_42)")
}

func TestDeviceLogin_EmptyOrgFails(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	srv := deviceFlowServer(t, "", "james@unkey.com") // no org selected
	defer srv.Close()

	var out bytes.Buffer
	client := oauth.New(oauth.WithBaseURL(srv.URL))
	err := deviceLogin(context.Background(), client, "client_abc", &out, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "organization")

	// Nothing should have been persisted.
	path, _ := cli.UserConfigPath()
	cfg, _ := cli.LoadUserConfig(path)
	require.False(t, cfg.HasOAuth())
}

func TestDeviceLogin_BrowserOpenFailureStillSucceeds(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	srv := deviceFlowServer(t, "org_1", "e@x.com")
	defer srv.Close()

	var out bytes.Buffer
	client := oauth.New(oauth.WithBaseURL(srv.URL))
	// openURL panics/does nothing; login must not depend on it.
	err := deviceLogin(context.Background(), client, "client_abc", &out, func(string) {})
	require.NoError(t, err)
	require.Contains(t, out.String(), "https://work.os/device?code=WDJB-MJHT")
}
