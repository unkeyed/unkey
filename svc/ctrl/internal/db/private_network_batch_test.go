package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestFindPrivateNetworkDeploymentIdsAnswersEveryKeyInOrder guarantees that
// batched lookups keep each answer with its key when the keys span several
// statements.
func TestFindPrivateNetworkDeploymentIdsAnswersEveryKeyInOrder(t *testing.T) {
	q := NewQueries(beginRollbackTx(t, openTestDatabase(t)))
	workspaceID, projectID, appID := uid.New(uid.WorkspacePrefix), uid.New(uid.ProjectPrefix), uid.New(uid.AppPrefix)
	previewID, emptyPreviewID := uid.New(uid.EnvironmentPrefix), uid.New(uid.EnvironmentPrefix)
	insertTestWorkspace(t, q, workspaceID, uid.DNS1035())
	insertTestProject(t, q, workspaceID, projectID)
	insertTestApp(t, q, workspaceID, projectID, appID, "app")
	insertTestEnvironment(t, q, workspaceID, projectID, appID, previewID, "preview", mysqltype.EnvironmentKindPreview)
	insertTestEnvironment(t, q, workspaceID, projectID, appID, emptyPreviewID, "empty", mysqltype.EnvironmentKindPreview)

	ready := sql.NullInt64{Int64: 1, Valid: true}
	oldID, newestID, neverReadyID := uid.New(uid.DeploymentPrefix), uid.New(uid.DeploymentPrefix), uid.New(uid.DeploymentPrefix)
	for _, deployment := range []testDeployment{
		{ID: oldID, GitBranch: "main", FirstReadyAt: ready, CreatedAt: 1},
		{ID: newestID, GitBranch: "main", FirstReadyAt: ready, CreatedAt: 2},
		{ID: neverReadyID, GitBranch: "feature", CreatedAt: 3},
	} {
		deployment.WorkspaceID, deployment.ProjectID, deployment.AppID, deployment.EnvironmentID = workspaceID, projectID, appID, previewID
		deployment.Status = mysqltype.DeploymentsStatusReady
		insertTestDeployment(t, q, deployment)
	}

	branch := func(name string) FindPrivateNetworkPreviewDeploymentIdParams {
		return FindPrivateNetworkPreviewDeploymentIdParams{
			AppID: appID, GitBranch: sql.NullString{String: name, Valid: true}, ForkKey: sql.NullString{String: "", Valid: true},
			WorkspaceID: workspaceID, ProjectID: projectID,
		}
	}
	var previews []FindPrivateNetworkPreviewDeploymentIdParams
	var wantPreviews []string
	for i := range 2*privateNetworkLookupChunkSize + 1 {
		switch i % 3 {
		case 0:
			previews, wantPreviews = append(previews, branch("main")), append(wantPreviews, newestID)
		case 1:
			previews, wantPreviews = append(previews, branch("feature")), append(wantPreviews, "")
		default:
			previews, wantPreviews = append(previews, branch(uid.New("branch"))), append(wantPreviews, "")
		}
	}
	gotPreviews, err := q.FindPrivateNetworkPreviewDeploymentIds(t.Context(), previews)
	require.NoError(t, err)
	require.Equal(t, wantPreviews, gotPreviews)

	environment := func(id string) FindPrivateNetworkEnvironmentDeploymentIdParams {
		return FindPrivateNetworkEnvironmentDeploymentIdParams{AppID: appID, EnvironmentID: id, WorkspaceID: workspaceID, ProjectID: projectID}
	}
	gotEnvironments, err := q.FindPrivateNetworkEnvironmentDeploymentIds(t.Context(), []FindPrivateNetworkEnvironmentDeploymentIdParams{
		environment(emptyPreviewID), environment(previewID),
	})
	require.NoError(t, err)
	require.Equal(t, []string{"", newestID}, gotEnvironments)

	none, err := q.FindPrivateNetworkEnvironmentDeploymentIds(t.Context(), nil)
	require.NoError(t, err)
	require.Empty(t, none)
}
