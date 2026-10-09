package handler_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/projects"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

type seededNamespace struct {
	id, projectID, name string
}

func seedNamespace(t *testing.T, h *testutil.Harness, workspaceID, name string) seededNamespace {
	t.Helper()
	projectID, err := projects.EnsureDefaultProject(t.Context(), h.DB.RW(), workspaceID)
	require.NoError(t, err)
	id := uid.New(uid.RatelimitNamespacePrefix)
	require.NoError(t, db.Query.InsertRatelimitNamespace(t.Context(), h.DB.RW(), db.InsertRatelimitNamespaceParams{
		ID:          id,
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		Name:        name,
		CreatedAt:   time.Now().UnixMilli(),
	}))
	return seededNamespace{id: id, projectID: projectID, name: name}
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
}
