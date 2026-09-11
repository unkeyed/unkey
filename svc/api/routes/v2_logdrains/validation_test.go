package logdrains_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	logdrains "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains"
)

func TestLogdrainsRejectInvalidInput(t *testing.T) {
	h := testutil.NewHarness(t)
	workspaceID := h.Resources().UserWorkspace.ID
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":**#*")
	headers := http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}
	create := &logdrains.Create{DB: h.DB, Vault: h.Vault, Auditlogs: h.Auditlogs, Clock: h.Clock}
	update := &logdrains.Update{DB: h.DB, Vault: h.Vault, Auditlogs: h.Auditlogs, Clock: h.Clock}
	get := &logdrains.Get{DB: h.DB}
	list := &logdrains.List{DB: h.DB}
	remove := &logdrains.Delete{DB: h.DB, Auditlogs: h.Auditlogs}
	metrics := &logdrains.Metrics{DB: h.DB, ClickHouse: h.ClickHouse, Clock: h.Clock}
	deliveries := &logdrains.Deliveries{DB: h.DB, ClickHouse: h.ClickHouse, Clock: h.Clock}
	for _, route := range []zen.Route{create, update, get, list, remove, metrics, deliveries} {
		h.Register(route)
	}
	for _, tc := range []struct {
		name  string
		route zen.Route
		body  string
	}{
		{"missing create fields", create, `{}`},
		{"empty name", create, `{"name":" ","stream":"audit_logs","destination":{"http":{"url":"https://logs.example.com"}}}`},
		{"zero batch size", create, `{"name":"Logs","stream":"audit_logs","batchSize":0,"destination":{"http":{"url":"https://logs.example.com"}}}`},
		{"mismatched filter", create, `{"name":"Logs","stream":"audit_logs","filters":{"passed":[false]},"destination":{"http":{"url":"https://logs.example.com"}}}`},
		{"empty filter value", create, `{"name":"Logs","stream":"audit_logs","filters":{"eventTypes":[" "]},"destination":{"http":{"url":"https://logs.example.com"}}}`},
		{"two destinations", create, `{"name":"Logs","stream":"audit_logs","destination":{"http":{"url":"https://logs.example.com"},"axiom":{"dataset":"logs","token":"secret"}}}`},
		{"URL credentials", create, `{"name":"Logs","stream":"audit_logs","destination":{"http":{"url":"https://user:secret@logs.example.com"}}}`},
		{"duplicate header names", create, `{"name":"Logs","stream":"audit_logs","destination":{"http":{"url":"https://logs.example.com","headers":[{"name":"Authorization","mode":"set","value":"secret"},{"name":"authorization","mode":"set","value":"other"}]}}}`},
		{"preserve on create", create, `{"name":"Logs","stream":"audit_logs","destination":{"http":{"url":"https://logs.example.com","headers":[{"name":"Authorization","mode":"preserve"}]}}}`},
		{"get requires ID", get, `{}`},
		{"list is bounded", list, `{"limit":101}`},
		{"update requires change", update, `{"logdrainId":"ld_example"}`},
		{"failure status is internal", update, `{"logdrainId":"ld_example","status":"paused_by_failure"}`},
		{"delete requires ID", remove, `{}`},
		{"metrics window is bounded", metrics, `{"logdrainId":"ld_example","hours":2}`},
		{"deliveries requires ID", deliveries, `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := testutil.CallRoute[json.RawMessage, json.RawMessage](h, tc.route, headers, json.RawMessage(tc.body))
			require.Equal(t, http.StatusBadRequest, result.Status, "%s", result.RawBody)
			require.Contains(t, string(result.RawBody), "application/invalid_input")
		})
	}
}
