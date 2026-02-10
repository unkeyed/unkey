package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
)

const testRepoFullName = "acme/repo"

func TestGitHubWebhook_Push_TriggersDeployWorkflow(t *testing.T) {
	deployRequests := make(chan *hydrav1.DeployRequest, 1)
	harness := newWebhookHarness(t, webhookHarnessConfig{
		Services: []restate.ServiceDefinition{hydrav1.NewDeployServiceServer(&mockDeployService{requests: deployRequests})},
	})
	projectID := insertRepoConnection(t, harness, testRepoFullName, 101, 202)

	resp, err := sendWebhook(fmt.Sprintf("%s/webhooks/github", harness.CtrlURL), webhookBody(testRepoFullName), harness.Secret)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()

	select {
	case req := <-deployRequests:
		require.NotEmpty(t, req.GetDeploymentId())
		gitSource := req.GetGit()
		require.NotNil(t, gitSource, "expected GitSource in deploy request")
		require.Equal(t, int64(101), gitSource.GetInstallationId())
		require.Equal(t, testRepoFullName, gitSource.GetRepository())
		require.Equal(t, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", gitSource.GetCommitSha())
		_ = projectID // projectID is stored in the deployment record, not passed to workflow
	case <-time.After(10 * time.Second):
		t.Fatal("expected deploy workflow invocation")
	}
}

func TestGitHubWebhook_InvalidSignature(t *testing.T) {
	deployRequests := make(chan *hydrav1.DeployRequest, 1)
	harness := newWebhookHarness(t, webhookHarnessConfig{
		Services: []restate.ServiceDefinition{hydrav1.NewDeployServiceServer(&mockDeployService{requests: deployRequests})},
	})
	_ = insertRepoConnection(t, harness, testRepoFullName, 101, 202)

	resp, err := sendWebhook(fmt.Sprintf("%s/webhooks/github", harness.CtrlURL), webhookBody(testRepoFullName), "wrong-secret")
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	_ = resp.Body.Close()

	select {
	case <-deployRequests:
		t.Fatal("unexpected deploy workflow invocation")
	case <-time.After(1 * time.Second):
	}
}

func insertRepoConnection(t *testing.T, harness *webhookHarness, repoFullName string, installationID, repositoryID int64) string {
	t.Helper()

	projectID := uid.New("prj")
	project := harness.Seed.CreateProject(harness.ctx, seed.CreateProjectRequest{
		ID:               projectID,
		WorkspaceID:      harness.Seed.Resources.UserWorkspace.ID,
		Name:             "test-project",
		Slug:             uid.New("slug"),
		GitRepositoryURL: fmt.Sprintf("https://github.com/%s", repoFullName),
		DefaultBranch:    "main",
		DeleteProtection: false,
	})

	// Create production environment (required for webhook handler to find environment by slug)
	harness.Seed.CreateEnvironment(harness.ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New("env"),
		WorkspaceID:      harness.Seed.Resources.UserWorkspace.ID,
		ProjectID:        project.ID,
		Slug:             "production",
		Description:      "",
		SentinelConfig:   []byte("{}"),
		DeleteProtection: false,
	})

	createdAt := time.Now().UnixMilli()
	params := db.InsertGithubRepoConnectionParams{
		ProjectID:          project.ID,
		InstallationID:     installationID,
		RepositoryID:       repositoryID,
		RepositoryFullName: repoFullName,
		CreatedAt:          createdAt,
		UpdatedAt:          sql.NullInt64{Valid: false},
	}
	require.NoError(t, db.Query.InsertGithubRepoConnection(harness.ctx, harness.DB.RW(), params))

	return project.ID
}

func sendWebhook(url string, body []byte, secret string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-Hub-Signature-256", sign(body, secret))

	client := &http.Client{Timeout: 10 * time.Second}
	return client.Do(req)
}

func webhookBody(repoFullName string) []byte {
	return []byte(fmt.Sprintf(`{
		"ref": "refs/heads/main",
		"after": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"installation": {"id": 101},
		"repository": {"id": 202, "full_name": "%s"},
		"commits": [{"id": "c1", "message": "m", "timestamp": "2024-01-01T00:00:00Z", "author": {"name": "n", "username": "u"}}],
		"head_commit": {"id": "c1", "message": "hello\nworld", "timestamp": "2024-01-01T00:00:00Z", "author": {"name": "n", "username": "u"}},
		"sender": {"login": "u", "avatar_url": "https://avatar"}
	}`, repoFullName))
}

func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
