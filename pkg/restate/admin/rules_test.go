package admin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpsertBuildConcurrencyRules(t *testing.T) {
	var got []rule
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		require.Equal(t, "/limits/rules", r.URL.Path)
		require.Equal(t, "Bearer secret", r.Header.Get("Authorization"))
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	err := New(Config{BaseURL: server.URL, APIKey: "secret"}).UpsertBuildConcurrencyRules(context.Background(), "ws_123", 4)
	require.NoError(t, err)
	require.Equal(t, []rule{
		{Pattern: "ws_123", Description: "Unkey workspace build concurrency", Limits: ruleLimits{Concurrency: 4}, Precondition: precondition{Type: "none"}},
		{Pattern: "ws_123/preview", Description: "Unkey preview build concurrency", Limits: ruleLimits{Concurrency: 3}, Precondition: precondition{Type: "none"}},
	}, got)
}

func TestUpsertDefaultBuildConcurrencyRule(t *testing.T) {
	var got []rule
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
	}))
	t.Cleanup(server.Close)

	require.NoError(t, New(Config{BaseURL: server.URL}).UpsertDefaultBuildConcurrencyRule(context.Background()))
	require.Equal(t, []rule{{
		Pattern: "*", Description: "Unkey default workspace build concurrency", Limits: ruleLimits{Concurrency: 1}, Precondition: precondition{Type: "none"},
	}}, got)
}

func TestUpsertBuildConcurrencyRulesMinimumPreviewLimit(t *testing.T) {
	var got []rule
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
	}))
	t.Cleanup(server.Close)

	require.NoError(t, New(Config{BaseURL: server.URL}).UpsertBuildConcurrencyRules(context.Background(), "ws", 1))
	require.Equal(t, int32(1), got[1].Limits.Concurrency)
}

func TestUpsertBuildConcurrencyRulesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, strings.Repeat("x", 70<<10))
	}))
	t.Cleanup(server.Close)

	err := New(Config{BaseURL: server.URL}).UpsertBuildConcurrencyRules(context.Background(), "ws", 1)
	require.ErrorContains(t, err, "status 400")
	require.Less(t, len(err.Error()), 65<<10)
}
