package logdrains_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	logdrains "github.com/unkeyed/unkey/svc/api/routes/v2_logdrains_create_logdrain"
	"github.com/unkeyed/unkey/svc/logdrain/sink/httpdrain"
	"google.golang.org/protobuf/proto"
)

func TestCreateHTTPFormatSupportsDelivery(t *testing.T) {
	for _, tc := range []struct {
		name   string
		format *openapi.LogdrainHttpWriteFormat
		want   logdrainv1.HttpBodyFormat
	}{
		{name: "default", want: logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_JSON},
		{name: "hec", format: new(openapi.LogdrainHttpWriteFormat("hec")), want: logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_HEC},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := testutil.NewHarness(t)
			route := &logdrains.Create{DB: h.DB, Vault: h.Vault, Auditlogs: h.Auditlogs, Clock: h.Clock, LimitsCache: h.Caches.WorkspaceLimits}
			h.Register(route)
			workspaceID := h.Resources().UserWorkspace.ID
			_, err := h.DB.RW().ExecContext(t.Context(), "UPDATE `limits` SET logdrains_max = 1 WHERE workspace_id = ?", workspaceID)
			require.NoError(t, err)
			key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":logdrains/*#write")
			response := testutil.CallRoute[openapi.CreateLogdrainRequest, openapi.LogdrainMutationResponse](h, route, http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}, openapi.CreateLogdrainRequest{
				Name: "HTTP logs", Stream: "audit_logs",
				Destination: openapi.LogdrainDestinationWrite{Http: &openapi.LogdrainHttpWrite{Url: new("https://logs.example.com"), Format: tc.format}},
			})
			require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
			var stored []byte
			require.NoError(t, h.DB.RW().QueryRowContext(t.Context(), "SELECT config FROM logdrains WHERE id = ?", response.Body.Data.Id).Scan(&stored))
			config := &logdrainv1.Config{}
			require.NoError(t, proto.Unmarshal(stored, config))
			_, err = httpdrain.New(httpdrain.Config{Endpoint: config.GetHttp().GetUrl(), Format: config.GetHttp().GetFormat()})
			require.NoError(t, err)
			require.Equal(t, tc.want, config.GetHttp().GetFormat())
		})
	}
}

func TestCreatePersistsEncryptedDestination(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &logdrains.Create{DB: h.DB, Vault: h.Vault, Auditlogs: h.Auditlogs, Clock: h.Clock, LimitsCache: h.Caches.WorkspaceLimits}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	_, err := h.DB.RW().ExecContext(context.Background(), "UPDATE `limits` SET logdrains_max = 1 WHERE workspace_id = ?", workspaceID)
	require.NoError(t, err)
	key := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":logdrains/*#write")
	headers := http.Header{"Authorization": {"Bearer " + key}, "Content-Type": {"application/json"}}
	response := testutil.CallRoute[openapi.CreateLogdrainRequest, openapi.LogdrainMutationResponse](h, route, headers, openapi.CreateLogdrainRequest{
		Name: "My logs", Stream: "audit_logs", BatchSize: new(int64(17)),
		Destination: openapi.LogdrainDestinationWrite{Axiom: &openapi.LogdrainAxiomWrite{Dataset: new("production"), Token: new("secret-token")}},
		Filters:     &openapi.LogdrainFilters{EventTypes: new([]string{"  key.create  "})},
	})
	require.Equal(t, http.StatusOK, response.Status, "%s", response.RawBody)
	require.NotEmpty(t, response.Body.Data.Id)
	require.NotEmpty(t, response.Body.Meta.RequestId)
	require.NotContains(t, string(response.RawBody), "secret-token")
	var stored []byte
	var owner, name, stream, status string
	var cursor, createdAt int64
	require.NoError(t, h.DB.RW().QueryRowContext(context.Background(), "SELECT workspace_id, name, stream, status, config, committed_offset_inserted_at, created_at FROM logdrains WHERE id = ?", response.Body.Data.Id).Scan(&owner, &name, &stream, &status, &stored, &cursor, &createdAt))
	require.Equal(t, workspaceID, owner)
	require.Equal(t, "My logs", name)
	require.Equal(t, "audit_logs", stream)
	require.Equal(t, "running", status)
	require.Equal(t, h.Clock.Now().UnixMilli(), createdAt)
	require.Equal(t, createdAt, cursor)
	require.NotContains(t, string(stored), "secret-token")
	config := &logdrainv1.Config{}
	require.NoError(t, proto.Unmarshal(stored, config))
	require.Equal(t, uint32(17), config.GetBatchSize())
	require.Equal(t, "production", config.GetAxiom().GetDataset())
	require.Equal(t, []string{"key.create"}, config.GetAuditLogs().GetEventTypes())
	decrypted, err := h.Vault.Decrypt(context.Background(), &vaultv1.DecryptRequest{Keyring: workspaceID, Encrypted: config.GetAxiom().GetEncryptedToken()})
	require.NoError(t, err)
	require.Equal(t, "secret-token", decrypted.GetPlaintext())
	events := h.FindAuditLogsByTargetID(context.Background(), t, response.Body.Data.Id)
	require.Len(t, events, 1)
	require.Equal(t, "logdrain.create", events[0].Event)
}
