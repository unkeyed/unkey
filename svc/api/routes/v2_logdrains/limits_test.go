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
	logdrains "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains"
)

func TestConcurrentCreatesRespectAllowance(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.Create{DB: h.DB, Vault: h.Vault, Auditlogs: h.Auditlogs, Clock: h.Clock}
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
	require.Equal(t, map[int]int{http.StatusOK: 2, http.StatusForbidden: 3, http.StatusBadRequest: 2}, counts)
	_, err = h.DB.RW().ExecContext(context.Background(), "UPDATE `limits` SET logdrains_max = 0 WHERE workspace_id = ?", workspaceID)
	require.NoError(t, err)
	response := testutil.CallRoute[json.RawMessage, openapi.LogdrainMutationResponse](h, route, headers, input)
	require.Equal(t, http.StatusForbidden, response.Status)
	require.Contains(t, response.RawBody, "Contact support to enable log drains")
}
