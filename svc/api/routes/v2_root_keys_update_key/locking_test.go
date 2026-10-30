package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_root_keys_update_key"
)

func TestRootKeyMutationPreservesDatabaseErrors(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	key := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
	route := newRoute(h)
	session := mutationSession(t, workspace.ID, key.KeyID, key.KeyID)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := route.Handle(ctx, session)
	require.ErrorIs(t, err, context.Canceled)
	_, coded := fault.GetCode(err)
	require.False(t, coded, "cancellation must not become an authorization failure")
}

// Reciprocal requests exercise the lock ordering in the handler itself.
func TestReciprocalRootKeyMutationsDoNotDeadlock(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	workspace := h.Resources().UserWorkspace
	a := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
	b := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	t.Cleanup(cancel)
	for range 32 {
		start := make(chan struct{})
		ready := make(chan struct{}, 2)
		results := make(chan error, 2)
		for _, pair := range [][2]string{{a.KeyID, b.KeyID}, {b.KeyID, a.KeyID}} {
			session := mutationSession(t, workspace.ID, pair[0], pair[1])
			go func() {
				ready <- struct{}{}
				select {
				case <-start:
					results <- route.Handle(ctx, session)
				case <-ctx.Done():
					results <- ctx.Err()
				}
			}()
		}
		for range 2 {
			select {
			case <-ready:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		}
		close(start)
		first, second := <-results, <-results
		require.NoError(t, first)
		require.NoError(t, second)
	}
	for _, pair := range [][2]string{{a.KeyID, b.KeyID}, {b.KeyID, a.KeyID}} {
		stored, err := db.Query.FindUnkeyRootKeyByID(ctx, h.DB.RO(), pair[1])
		require.NoError(t, err)
		require.Equal(t, pair[0], stored.Name.String)
	}
}

func mutationSession(t *testing.T, workspaceID, callerID, targetID string) *zen.Session {
	t.Helper()
	body, err := json.Marshal(handler.Request{KeyId: targetID, Name: nullable.NewNullableWithValue(callerID)})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/v2/rootKeys.updateKey", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	session := &zen.Session{}
	require.NoError(t, session.Init(httptest.NewRecorder(), request, 0))
	session.SetPrincipal(&principal.Principal{
		Type:                  principal.TypeAPIKey,
		Subject:               principal.Subject{ID: callerID, Type: principal.SubjectTypeRootKey},
		Source:                principal.KeySource{KeyID: callerID},
		AuthorizedWorkspaceID: workspaceID,
		Permissions:           []string{"unkey:v1:" + workspaceID + ":rootKeys/*#write"},
	})
	return session
}
