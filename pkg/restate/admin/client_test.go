package admin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestFindLiveInvocations(t *testing.T) {
	var gotAccept, gotQuery string
	aliveInvocationID1 := uid.New("inv")
	aliveInvocationID2 := uid.New("inv")
	deadInvocationID := uid.New("inv")
	type invocationRow struct {
		ID string `json:"id"`
	}
	response, err := json.Marshal(struct {
		Rows []invocationRow `json:"rows"`
	}{Rows: []invocationRow{{ID: aliveInvocationID1}, {ID: aliveInvocationID2}}})
	require.NoError(t, err)

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
		_, _ = w.Write(response)
	}))
	t.Cleanup(server.Close)

	client := New(Config{BaseURL: server.URL, APIKey: ""})

	live, err := client.FindLiveInvocations(context.Background(), []string{aliveInvocationID1, aliveInvocationID2, deadInvocationID})
	require.NoError(t, err)

	// The endpoint compares the Accept header literally. Any value other
	// than "application/json" returns binary Arrow.
	require.Equal(t, "application/json", gotAccept)
	require.Contains(t, gotQuery, "sys_invocation")
	require.Contains(t, gotQuery, "'"+deadInvocationID+"'")
	// concat(id, '') prevents Restate from parsing the literals as
	// invocation IDs; an unparseable ID would fail the whole query.
	require.Contains(t, gotQuery, "concat(id, '')")
	// Killed and cancelled invocations keep a row with status 'completed'
	// until retention expires. They must not count as live.
	require.Contains(t, gotQuery, "status <> 'completed'")

	require.Equal(t, map[string]bool{
		aliveInvocationID1: true,
		aliveInvocationID2: true,
		deadInvocationID:   false,
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
