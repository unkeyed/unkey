package handler_test

import (
	"fmt"
	"testing"

	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
)

func createTestProject(t *testing.T, h *testutil.Harness, workspaceID string) string {
	t.Helper()

	projectID := uid.New(uid.ProjectPrefix)
	h.CreateProject(seed.CreateProjectRequest{
		ID:          projectID,
		WorkspaceID: workspaceID,
		Name:        uid.New("test"),
		Slug:        uid.New("test"),
	})
	return projectID
}

func keyspaceGrant(workspaceID, projectID, keyspaceID, action string) string {
	return fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s#%s", workspaceID, projectID, keyspaceID, action)
}

func keyGrant(workspaceID, projectID, keyspaceID, action string) string {
	return fmt.Sprintf("unkey:v1:%s:projects/%s/keyspaces/%s/keys/*#%s", workspaceID, projectID, keyspaceID, action)
}
