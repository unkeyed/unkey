package github

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type tunnelTransport func(*http.Request) (*http.Response, error)

func (transport tunnelTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return transport(req)
}

func TestTunnelUsesChildOutputInsteadOfLocalAdminAPI(t *testing.T) {
	tests := []struct {
		name       string
		address    string
		publicURL  string
		webhookURL string
	}{
		{name: "child tunnel", address: "http://localhost:7091", publicURL: "https://child.ngrok.app", webhookURL: "https://child.ngrok.app/webhooks/github"},
		{name: "wrong port", address: "http://localhost:8080", publicURL: "https://child.ngrok.app"},
		{name: "non HTTPS", address: "http://localhost:7091", publicURL: "http://child.ngrok.app"},
		{name: "missing host", address: "http://localhost:7091", publicURL: "https:///missing"},
		{name: "credentials", address: "http://localhost:7091", publicURL: "https://user@child.ngrok.app"},
		{name: "query", address: "http://localhost:7091", publicURL: "https://child.ngrok.app?redirect=other"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testTunnelDiscovery(t, test.address, test.publicURL, test.webhookURL)
		})
	}
}

func testTunnelDiscovery(t *testing.T, address, publicURL, expectedWebhookURL string) {
	t.Helper()
	directory := t.TempDir()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	keyPath := filepath.Join(directory, "key.pem")
	require.NoError(t, os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{
		Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key),
	}), 0o600))
	envPath := filepath.Join(directory, "app.env")
	require.NoError(t, os.WriteFile(envPath, []byte("UNKEY_GITHUB_APP_ID=1\n"), 0o600))
	log, err := json.Marshal(struct {
		Message string `json:"msg"`
		Address string `json:"addr"`
		URL     string `json:"url"`
	}{Message: "started tunnel", Address: address, URL: publicURL})
	require.NoError(t, err)
	t.Setenv("NGROK_TEST_LOG", string(log))
	pidPath := filepath.Join(directory, "pid")
	t.Setenv("NGROK_TEST_PID", pidPath)
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
	script := "#!/bin/sh\necho $$ > \"$NGROK_TEST_PID\"\nprintf '%s\\n' \"$NGROK_TEST_LOG\"\n"
	if expectedWebhookURL != "" {
		script += "exec sleep 60\n"
	}
	require.NoError(t, os.WriteFile(filepath.Join(directory, "ngrok"), []byte(script), 0o700))
	killChild := func() {
		data, readErr := os.ReadFile(pidPath)
		require.NoError(t, readErr)
		pid, parseErr := strconv.Atoi(strings.TrimSpace(string(data)))
		require.NoError(t, parseErr)
		process, findErr := os.FindProcess(pid)
		require.NoError(t, findErr)
		_ = process.Kill()
	}
	t.Cleanup(killChild)
	transport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = transport })
	webhookURL := ""
	http.DefaultTransport = tunnelTransport(func(req *http.Request) (*http.Response, error) {
		body := ""
		if req.URL.Host == "api.github.com" {
			var config struct {
				URL string `json:"url"`
			}
			require.NoError(t, json.NewDecoder(req.Body).Decode(&config))
			webhookURL = config.URL
			killChild()
		} else {
			response := struct {
				Tunnels []struct {
					PublicURL string `json:"public_url"`
					Proto     string `json:"proto"`
				} `json:"tunnels"`
			}{}
			response.Tunnels = append(response.Tunnels, struct {
				PublicURL string `json:"public_url"`
				Proto     string `json:"proto"`
			}{PublicURL: "https://attacker.example", Proto: "https"})
			encoded, encodeErr := json.Marshal(response)
			require.NoError(t, encodeErr)
			body = string(encoded)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	err = tunnelCmd.Run(context.Background(), []string{"tunnel", "--env-file", envPath, "--pem-file", keyPath})
	if expectedWebhookURL == "" {
		require.ErrorContains(t, err, "ngrok output ended")
	} else {
		require.NoError(t, err)
	}
	require.Equal(t, expectedWebhookURL, webhookURL)
}
