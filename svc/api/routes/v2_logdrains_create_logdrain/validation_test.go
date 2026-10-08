package logdrains_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/openapi/validation"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	logdrains "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains_create_logdrain"
	"google.golang.org/protobuf/proto"
)

func TestCreateSchemaRequiresDestinationFields(t *testing.T) {
	validator, err := validation.NewFromBytes(openapi.Spec)
	require.NoError(t, err)
	for _, tc := range []struct {
		name        string
		destination string
		valid       bool
	}{
		{"missing HTTP URL", `{"http":{"format":"json"}}`, false},
		{"missing Axiom dataset", `{"axiom":{"token":"secret"}}`, false},
		{"missing Axiom token", `{"axiom":{"dataset":"logs"}}`, false},
		{"HTTP URL", `{"http":{"url":"https://logs.example.com"}}`, true},
		{"Axiom dataset and token", `{"axiom":{"dataset":"logs","token":"secret"}}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/v2/logdrains.createLogdrain", strings.NewReader(`{"name":"Logs","stream":{"auditLogs":{}},"destination":`+tc.destination+`}`))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer test")
			result := validator.Validate(request)
			if tc.valid {
				require.Nil(t, result)
			} else {
				require.NotNil(t, result)
			}
		})
	}
}

func TestCreateSchemaAcceptsNestedStream(t *testing.T) {
	validator, err := validation.NewFromBytes(openapi.Spec)
	require.NoError(t, err)
	for _, tc := range []struct {
		name   string
		stream string
		valid  bool
	}{
		{"boolean array", `{"ratelimits":{"passed":[false]}}`, true},
		{"unfiltered stream", `{"auditLogs":{}}`, true},
		{"numeric status class", `{"gatewayRequests":{"statusClasses":[2]}}`, false},
		{"unsupported status class", `{"gatewayRequests":{"statusClasses":["6xx"]}}`, false},
		{"no stream", `{}`, false},
		{"multiple streams", `{"auditLogs":{},"ratelimits":{}}`, false},
		{"mismatched filters", `{"auditLogs":{"passed":[false]}}`, false},
		{"scalar passed", `{"ratelimits":{"passed":false}}`, false},
		{"unknown stream", `{"other":{}}`, false},
		{"legacy stream", `"audit_logs"`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/v2/logdrains.createLogdrain", strings.NewReader(`{"name":"Logs","stream":`+tc.stream+`,"destination":{"http":{"url":"https://logs.example.com"}}}`))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer test")
			if tc.valid {
				require.Nil(t, validator.Validate(request))
			} else {
				require.NotNil(t, validator.Validate(request))
			}
		})
	}
}

func TestCreateRejectsInvalidInputWithoutChangingExistingDrains(t *testing.T) {
	h := testutil.NewHarness(t)
	workspaceID := h.Resources().UserWorkspace.ID
	id := uid.New("ld")
	config, err := proto.Marshal(&logdrainv1.Config{
		Destination: &logdrainv1.Config_Http{
			Http: &logdrainv1.HttpConfig{Url: "https://logs.example.com"},
		},
	})
	require.NoError(t, err)
	_, err = h.DB.RW().ExecContext(context.Background(), "INSERT INTO logdrains (id, workspace_id, name, stream, config, lease_id, fencing_token, created_at) VALUES (?, ?, 'Unchanged', 'audit_logs', ?, '', '', 123)", id, workspaceID, config)
	require.NoError(t, err)
	_, err = h.DB.RW().ExecContext(context.Background(), "UPDATE `limits` SET logdrains_max = 2 WHERE workspace_id = ?", workspaceID)
	require.NoError(t, err)
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":**#*")
	headers := http.Header{
		"Authorization": {"Bearer " + key},
		"Content-Type":  {"application/json"},
	}
	route := &logdrains.Handler{
		DB:          h.DB,
		Vault:       h.Vault,
		Auditlogs:   h.Auditlogs,
		Clock:       h.Clock,
		LimitsCache: h.Caches.WorkspaceLimits,
	}
	h.Register(route)
	for _, tc := range []struct {
		name string
		body string
	}{
		{"missing create fields", `{}`},
		{"missing HTTP URL", `{"name":"Logs","stream":{"auditLogs":{}},"destination":{"http":{"format":"json"}}}`},
		{"missing Axiom token", `{"name":"Logs","stream":{"auditLogs":{}},"destination":{"axiom":{"dataset":"logs"}}}`},
		{"missing Axiom dataset", `{"name":"Logs","stream":{"auditLogs":{}},"destination":{"axiom":{"token":"secret"}}}`},
		{"unsupported URL scheme", `{"name":"Logs","stream":{"auditLogs":{}},"destination":{"http":{"url":"ftp://logs.example.com"}}}`},
		{"unsupported HTTP format", `{"name":"Logs","stream":{"auditLogs":{}},"destination":{"http":{"url":"https://logs.example.com","format":"csv"}}}`},
		{"empty name", `{"name":" ","stream":{"auditLogs":{}},"destination":{"http":{"url":"https://logs.example.com"}}}`},
		{"zero batch size", `{"name":"Logs","stream":{"auditLogs":{}},"batchSize":0,"destination":{"http":{"url":"https://logs.example.com"}}}`},
		{"mismatched filter", `{"name":"Logs","stream":{"auditLogs":{"passed":[false]}},"destination":{"http":{"url":"https://logs.example.com"}}}`},
		{"empty filter value", `{"name":"Logs","stream":{"auditLogs":{"eventTypes":[" "]}},"destination":{"http":{"url":"https://logs.example.com"}}}`},
		{"two destinations", `{"name":"Logs","stream":{"auditLogs":{}},"destination":{"http":{"url":"https://logs.example.com"},"axiom":{"dataset":"logs","token":"secret"}}}`},
		{"URL credentials", `{"name":"Logs","stream":{"auditLogs":{}},"destination":{"http":{"url":"https://user:secret@logs.example.com"}}}`},
		{"duplicate header names", `{"name":"Logs","stream":{"auditLogs":{}},"destination":{"http":{"url":"https://logs.example.com","headers":[{"name":"Authorization","value":"secret"},{"name":"authorization","value":"other"}]}}}`},
		{"invalid header name", `{"name":"Logs","stream":{"auditLogs":{}},"destination":{"http":{"url":"https://logs.example.com","headers":[{"name":"Bad Name","value":"secret"}]}}}`},
		{"invalid header value", `{"name":"Logs","stream":{"auditLogs":{}},"destination":{"http":{"url":"https://logs.example.com","headers":[{"name":"Authorization","value":"secret\r\nInjected: value"}]}}}`},
		{"missing header value", `{"name":"Logs","stream":{"auditLogs":{}},"destination":{"http":{"url":"https://logs.example.com","headers":[{"name":"Authorization"}]}}}`},
		{"legacy top-level filters", `{"name":"Logs","stream":{"auditLogs":{}},"filters":{"eventTypes":[]},"destination":{"http":{"url":"https://logs.example.com"}}}`},
		{"preserve on create", `{"name":"Logs","stream":{"auditLogs":{}},"destination":{"http":{"url":"https://logs.example.com","headers":[{"name":"Authorization","mode":"preserve"}]}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := testutil.CallRoute[json.RawMessage, openapi.BadRequestErrorResponse](h, route, headers, json.RawMessage(tc.body))
			require.Equal(t, http.StatusBadRequest, result.Status, "%s", result.RawBody)
			require.Equal(t, http.StatusBadRequest, result.Body.Error.Status)
			require.Contains(t, result.Body.Error.Type, "application/invalid_input")
			require.NotEmpty(t, result.Body.Error.Detail)
			require.NotEmpty(t, result.Body.Meta.RequestId)
			var name, status string
			var stored []byte
			require.NoError(t, h.DB.RW().QueryRowContext(context.Background(), "SELECT name, status, config FROM logdrains WHERE id = ?", id).Scan(&name, &status, &stored))
			require.Equal(t, "Unchanged", name)
			require.Equal(t, "running", status)
			require.Equal(t, config, stored)
			require.Empty(t, h.FindAuditLogsByTargetID(context.Background(), t, id))
			var count int
			require.NoError(t, h.DB.RW().QueryRowContext(context.Background(), "SELECT COUNT(*) FROM logdrains WHERE workspace_id = ?", workspaceID).Scan(&count))
			require.Equal(t, 1, count)
		})
	}
}
