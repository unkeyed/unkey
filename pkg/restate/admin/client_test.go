package admin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindLiveInvocations(t *testing.T) {
	var gotAccept, gotQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/query", r.URL.Path)
		gotAccept = r.Header.Get("Accept")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var req struct {
			Query string `json:"query"`
		}
		require.NoError(t, json.Unmarshal(body, &req))
		gotQuery = req.Query

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"rows":[{"id":"inv_alive_1"},{"id":"inv_alive_2"}]}`))
	}))
	t.Cleanup(server.Close)

	client := New(Config{BaseURL: server.URL, APIKey: ""})

	live, err := client.FindLiveInvocations(context.Background(), []string{"inv_alive_1", "inv_alive_2", "inv_dead_1"})
	require.NoError(t, err)

	// The endpoint compares the Accept header literally; anything but
	// exactly "application/json" returns binary Arrow.
	require.Equal(t, "application/json", gotAccept)
	require.Contains(t, gotQuery, "sys_invocation")
	require.Contains(t, gotQuery, "'inv_dead_1'")

	require.Equal(t, map[string]bool{
		"inv_alive_1": true,
		"inv_alive_2": true,
		"inv_dead_1":  false,
	}, live)
}

func TestFindLiveInvocations_EmptyInput(t *testing.T) {
	client := New(Config{BaseURL: "http://unreachable.invalid", APIKey: ""})

	live, err := client.FindLiveInvocations(context.Background(), nil)
	require.NoError(t, err)
	require.Empty(t, live)
}

func TestFindLiveInvocations_RejectsMalformedIDs(t *testing.T) {
	client := New(Config{BaseURL: "http://unreachable.invalid", APIKey: ""})

	_, err := client.FindLiveInvocations(context.Background(), []string{"inv_ok", "bad'id"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid invocation id")
}

func TestFindLiveInvocations_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	client := New(Config{BaseURL: server.URL, APIKey: ""})

	_, err := client.FindLiveInvocations(context.Background(), []string{"inv_1"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "status 500")
}
