package clidevice

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestHTTPClientMetadataFetchesTheDocument(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "application/json", r.Header.Get("Accept"))
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(ClientMetadata{
			ClientID:                server.URL + r.URL.Path,
			ClientName:              "Example MCP",
			RedirectURIs:            []string{"https://client.example.com/callback"},
			TokenEndpointAuthMethod: "none",
		}))
	}))
	t.Cleanup(server.Close)
	clientID := server.URL + "/oauth/" + uid.New(uid.TestPrefix) + ".json"

	metadata, err := (&HTTPClientMetadata{Client: server.Client()}).Fetch(t.Context(), clientID)
	require.NoError(t, err)
	require.Equal(t, ClientMetadata{
		ClientID:                clientID,
		ClientName:              "Example MCP",
		RedirectURIs:            []string{"https://client.example.com/callback"},
		TokenEndpointAuthMethod: "none",
	}, metadata)
}

func TestHTTPClientMetadataRejectsUnusableResponses(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{name: "not found", handler: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}},
		{name: "redirect", handler: func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/elsewhere.json" {
				http.Redirect(w, r, "/elsewhere.json", http.StatusFound)
				return
			}
			require.NoError(t, json.NewEncoder(w).Encode(ClientMetadata{
				ClientID:                "https://client.example.com/oauth/client.json",
				ClientName:              "",
				RedirectURIs:            nil,
				TokenEndpointAuthMethod: "",
			}))
		}},
		{name: "oversized", handler: func(w http.ResponseWriter, _ *http.Request) {
			require.NoError(t, json.NewEncoder(w).Encode(ClientMetadata{
				ClientID:                "https://client.example.com/oauth/client.json",
				ClientName:              strings.Repeat("a", clientMetadataBytesMax),
				RedirectURIs:            nil,
				TokenEndpointAuthMethod: "",
			}))
		}},
		{name: "not json", handler: func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.Write([]byte("<html></html>"))
			require.NoError(t, err)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewTLSServer(tt.handler)
			t.Cleanup(server.Close)
			client := server.Client()
			client.CheckRedirect = NewHTTPClientMetadata().Client.CheckRedirect

			_, err := (&HTTPClientMetadata{Client: client}).Fetch(t.Context(), server.URL+"/oauth/client.json")
			require.Error(t, err)
		})
	}
}

func TestHTTPClientMetadataRefusesPrivateAddresses(t *testing.T) {
	requested := false
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requested = true
	}))
	t.Cleanup(server.Close)

	_, err := NewHTTPClientMetadata().Fetch(t.Context(), server.URL+"/oauth/client.json")
	require.Error(t, err)
	require.False(t, requested)
}
