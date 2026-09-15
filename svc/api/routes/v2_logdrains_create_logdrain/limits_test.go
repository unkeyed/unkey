package logdrains_test

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	logdrains "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains_create_logdrain"
)

func TestCreateUsesFreshCachedAllowance(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.Create{DB: h.DB, Vault: h.Vault, Auditlogs: h.Auditlogs, Clock: h.Clock, LimitsCache: h.Caches.WorkspaceLimits}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	_, err := h.DB.RW().ExecContext(context.Background(), "UPDATE `limits` SET logdrains_max = 2 WHERE workspace_id = ?", workspaceID)
	require.NoError(t, err)
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":logdrains/*#write")
	headers := http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}
	input := json.RawMessage(`{"name":"Cached allowance","stream":"audit_logs","destination":{"http":{"url":"https://logs.example.com"}}}`)
	first := testutil.CallRoute[json.RawMessage, openapi.LogdrainMutationResponse](h, route, headers, input)
	require.Equal(t, http.StatusOK, first.Status, "%s", first.RawBody)
	_, err = h.DB.RW().ExecContext(context.Background(), "UPDATE `limits` SET logdrains_max = 0 WHERE workspace_id = ?", workspaceID)
	require.NoError(t, err)
	second := testutil.CallRoute[json.RawMessage, openapi.LogdrainMutationResponse](h, route, headers, input)
	require.Equal(t, http.StatusOK, second.Status, "%s", second.RawBody)
	third := testutil.CallRoute[json.RawMessage, openapi.ForbiddenErrorResponse](h, route, headers, input)
	require.Equal(t, http.StatusForbidden, third.Status, "%s", third.RawBody)
	require.Contains(t, third.Body.Error.Detail, "increase this workspace's log drain allowance")
}

func TestCreateDeniesZeroOrMissingAllowance(t *testing.T) {
	for _, missing := range []bool{false, true} {
		name := "zero"
		if missing {
			name = "missing"
		}
		t.Run(name, func(t *testing.T) {
			h := testutil.NewHarness(t)
			route := &logdrains.Create{DB: h.DB, Vault: h.Vault, Auditlogs: h.Auditlogs, Clock: h.Clock, LimitsCache: h.Caches.WorkspaceLimits}
			h.Register(route)
			workspaceID := h.Resources().UserWorkspace.ID
			query := "UPDATE `limits` SET logdrains_max = 0 WHERE workspace_id = ?"
			if missing {
				query = "DELETE FROM `limits` WHERE workspace_id = ?"
			}
			_, err := h.DB.RW().ExecContext(context.Background(), query, workspaceID)
			require.NoError(t, err)
			key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":logdrains/*#write")
			headers := http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}
			response := testutil.CallRoute[json.RawMessage, openapi.ForbiddenErrorResponse](h, route, headers, json.RawMessage(`{"name":"Disabled","stream":"audit_logs","destination":{"http":{"url":"https://logs.example.com"}}}`))
			require.Equal(t, http.StatusForbidden, response.Status, "%s", response.RawBody)
			require.Contains(t, response.Body.Error.Detail, "Contact support to enable log drains")
			var count int
			require.NoError(t, h.DB.RW().QueryRowContext(context.Background(), "SELECT COUNT(*) FROM logdrains WHERE workspace_id = ?", workspaceID).Scan(&count))
			require.Zero(t, count)
		})
	}
}

func TestConcurrentCreatesValidateRequests(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.Create{DB: h.DB, Vault: h.Vault, Auditlogs: h.Auditlogs, Clock: h.Clock, LimitsCache: h.Caches.WorkspaceLimits}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	_, err := h.DB.RW().ExecContext(context.Background(), "UPDATE `limits` SET logdrains_max = 2 WHERE workspace_id = ?", workspaceID)
	require.NoError(t, err)
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":logdrains/*#write")
	headers := http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}
	input := json.RawMessage(`{"name":"Concurrent drain","stream":"audit_logs","destination":{"http":{"url":"https://logs.example.com"}}}`)
	start := make(chan struct{})
	statuses := make(chan int, 7)
	var workers sync.WaitGroup
	for range 5 {
		workers.Go(func() {
			<-start
			response := testutil.CallRoute[json.RawMessage, openapi.LogdrainMutationResponse](h, route, headers.Clone(), input)
			if response.Status != http.StatusOK && response.Status != http.StatusForbidden {
				t.Logf("Unexpected response: %s", response.RawBody)
			}
			statuses <- response.Status
		})
	}
	for _, invalid := range []json.RawMessage{
		json.RawMessage(`{"name":"Invalid mode","stream":"audit_logs","destination":{"http":{"url":"https://logs.example.com","headers":[{"name":"Authorization","mode":"bogus","value":"secret"}]}}}`),
		json.RawMessage(`{"name":"Missing header name","stream":"audit_logs","destination":{"http":{"url":"https://logs.example.com","headers":[{"mode":"set","value":"secret"}]}}}`),
	} {
		workers.Go(func() {
			<-start
			response := testutil.CallRoute[json.RawMessage, openapi.LogdrainMutationResponse](h, route, headers.Clone(), invalid)
			statuses <- response.Status
		})
	}
	close(start)
	workers.Wait()
	close(statuses)
	counts := map[int]int{}
	for status := range statuses {
		counts[status]++
	}
	require.Equal(t, 2, counts[http.StatusBadRequest])
	require.Equal(t, 5, counts[http.StatusOK]+counts[http.StatusForbidden])
	require.Positive(t, counts[http.StatusOK])
	var persisted int
	require.NoError(t, h.DB.RW().QueryRowContext(context.Background(), "SELECT COUNT(*) FROM logdrains WHERE workspace_id = ?", workspaceID).Scan(&persisted))
	require.Equal(t, counts[http.StatusOK], persisted)
}
