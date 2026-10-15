package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_update_key"
)

func TestUpdateKeyAuditsIdentityEndpoints(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB: h.DB, Auditlogs: h.Auditlogs,
		KeyCache: h.Caches.VerificationKeyByHash, UsageLimiter: h.UsageLimiter,
	}
	h.Register(route)
	workspaceID := h.Resources().UserWorkspace.ID
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID})
	oldIdentity := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspaceID, ProjectID: api.ProjectID,
		ExternalID: uid.New(uid.TestPrefix), Environment: "default",
	})
	newIdentity := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspaceID, ProjectID: api.ProjectID,
		ExternalID: uid.New(uid.TestPrefix), Environment: "default",
	})
	deletedIdentity := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspaceID, ProjectID: api.ProjectID,
		ExternalID: uid.New(uid.TestPrefix), Environment: "default",
	})
	require.NoError(t, db.Query.SoftDeleteIdentity(t.Context(), h.DB.RW(), db.SoftDeleteIdentityParams{
		WorkspaceID: workspaceID, IdentityID: deletedIdentity.ID,
	}))
	missingIdentityID := uid.New(uid.IdentityPrefix)
	rootKey := h.CreateRootKey(workspaceID, rootKeyGrant(workspaceID, api.ProjectID, api.KeyAuthID.String, "*", "write"))
	headers := http.Header{
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		"Content-Type":  {"application/json"},
	}

	for _, tt := range []struct {
		name               string
		initial            *string
		externalID         nullable.Nullable[string]
		identityID         string
		targetIDs          []string
		identityIDBefore   string
		identityNameBefore string
	}{
		{"assign", nil, nullable.NewNullableWithValue(newIdentity.ExternalID), newIdentity.ID, []string{newIdentity.ID}, "", ""},
		{"replace", &oldIdentity.ID, nullable.NewNullableWithValue(newIdentity.ExternalID), newIdentity.ID, []string{newIdentity.ID}, oldIdentity.ID, oldIdentity.ExternalID},
		{"remove", &oldIdentity.ID, nullable.NewNullNullable[string](), "", nil, oldIdentity.ID, oldIdentity.ExternalID},
		{"remove deleted identity", &deletedIdentity.ID, nullable.NewNullNullable[string](), "", nil, deletedIdentity.ID, deletedIdentity.ExternalID},
		{"remove missing identity", &missingIdentityID, nullable.NewNullNullable[string](), "", nil, missingIdentityID, ""},
		{"unchanged assignment", &oldIdentity.ID, nullable.NewNullableWithValue(oldIdentity.ExternalID), oldIdentity.ID, []string{oldIdentity.ID}, "", ""},
		{"omitted identity", &oldIdentity.ID, nullable.Nullable[string]{}, oldIdentity.ID, nil, "", ""},
		{"remove absent identity", nil, nullable.NewNullNullable[string](), "", nil, "", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			key := h.CreateKey(seed.CreateKeyRequest{
				WorkspaceID: workspaceID, KeySpaceID: api.KeyAuthID.String, IdentityID: tt.initial,
			})
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				KeyId: key.KeyID, ExternalId: tt.externalID, Enabled: new(true),
			})
			require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
			saved, err := db.Query.FindKeyByID(t.Context(), h.DB.RO(), key.KeyID)
			require.NoError(t, err)
			require.Equal(t, tt.identityID, saved.IdentityID.String)
			require.Equal(t, tt.identityID != "", saved.IdentityID.Valid)

			logs := h.FindAuditLogsByTargetID(t.Context(), t, key.KeyID)
			require.Len(t, logs, 1)
			require.Equal(t, "key.update", logs[0].Event)
			var identityIDs []string
			for _, target := range logs[0].Targets {
				if target.Type == "identity" {
					identityIDs = append(identityIDs, target.ID)
				}
				if target.Type == "key" {
					if tt.identityIDBefore == "" {
						require.Empty(t, target.Meta)
					} else {
						expected := map[string]any{"identityIdBefore": tt.identityIDBefore}
						if tt.identityNameBefore != "" {
							expected["identityNameBefore"] = tt.identityNameBefore
							expected["identityDisplayNameBefore"] = tt.identityNameBefore
						}
						require.Equal(t, expected, target.Meta)
					}
				}
			}
			require.ElementsMatch(t, tt.targetIDs, identityIDs)
			require.Len(t, logs[0].Targets, 2+len(tt.targetIDs))
		})
	}
}
