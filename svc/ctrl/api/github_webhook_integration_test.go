package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
)

const testRepoFullName = "acme/repo"

// mockGitHubWebhookService captures HandlePushRequests sent by the thin HTTP handler.
type mockGitHubWebhookService struct {
	hydrav1.UnimplementedGitHubWebhookServiceServer
	requests chan *hydrav1.HandlePushRequest
}

func (m *mockGitHubWebhookService) HandlePush(_ restate.ObjectContext, req *hydrav1.HandlePushRequest) (*hydrav1.HandlePushResponse, error) {
	m.requests <- req
	return &hydrav1.HandlePushResponse{}, nil
}

func TestGitHubWebhook_Push_TriggersHandlePush(t *testing.T) {
	pushRequests := make(chan *hydrav1.HandlePushRequest, 1)
	harness := newWebhookHarness(t, webhookHarnessConfig{
		Services: []restate.ServiceDefinition{hydrav1.NewGitHubWebhookServiceServer(&mockGitHubWebhookService{requests: pushRequests})},
	})

	resp, err := sendWebhook(fmt.Sprintf("%s/webhooks/github", harness.CtrlURL), mustMarshal(t, newTestPushPayload(testRepoFullName, false)), harness.Secret)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()

	select {
	case req := <-pushRequests:
		require.Equal(t, int64(101), req.GetInstallationId())
		require.Equal(t, int64(202), req.GetRepositoryId())
		require.Equal(t, testRepoFullName, req.GetRepositoryFullName())
		require.Equal(t, "main", req.GetBranch())
		require.Equal(t, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", req.GetAfter())
		require.Equal(t, "Merge pull request #1 from pr-creator/feat", req.GetCommitMessage())
		require.Equal(t, "merger", req.GetCommitAuthorHandle())
		require.Equal(t, "https://avatar", req.GetCommitAuthorAvatarUrl())
		require.NotZero(t, req.GetCommitTimestamp())
	case <-time.After(10 * time.Second):
		t.Fatal("expected HandlePush invocation")
	}
}

func TestGitHubWebhook_Push_ProcessesFork(t *testing.T) {
	pushRequests := make(chan *hydrav1.HandlePushRequest, 1)
	harness := newWebhookHarness(t, webhookHarnessConfig{
		Services: []restate.ServiceDefinition{hydrav1.NewGitHubWebhookServiceServer(&mockGitHubWebhookService{requests: pushRequests})},
	})

	resp, err := sendWebhook(fmt.Sprintf("%s/webhooks/github", harness.CtrlURL), mustMarshal(t, newTestPushPayload(testRepoFullName, true)), harness.Secret)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()

	select {
	case req := <-pushRequests:
		require.Equal(t, int64(101), req.GetInstallationId())
		require.Equal(t, int64(202), req.GetRepositoryId())
		require.Equal(t, testRepoFullName, req.GetRepositoryFullName())
	case <-time.After(10 * time.Second):
		t.Fatal("expected HandlePush invocation for fork with app installed")
	}
}

func TestGitHubWebhook_Push_IgnoresDeletedBranch(t *testing.T) {
	pushRequests := make(chan *hydrav1.HandlePushRequest, 1)
	harness := newWebhookHarness(t, webhookHarnessConfig{
		Services: []restate.ServiceDefinition{hydrav1.NewGitHubWebhookServiceServer(&mockGitHubWebhookService{requests: pushRequests})},
	})

	payload := newTestPushPayload(testRepoFullName, false)
	payload.Deleted = true
	payload.After = "0000000000000000000000000000000000000000"

	resp, err := sendWebhook(fmt.Sprintf("%s/webhooks/github", harness.CtrlURL), mustMarshal(t, payload), harness.Secret)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()

	select {
	case <-pushRequests:
		t.Fatal("unexpected HandlePush invocation for deleted branch")
	case <-time.After(1 * time.Second):
	}
}

func TestGitHubWebhook_Push_ProcessesCreatedBranch(t *testing.T) {
	pushRequests := make(chan *hydrav1.HandlePushRequest, 1)
	harness := newWebhookHarness(t, webhookHarnessConfig{
		Services: []restate.ServiceDefinition{hydrav1.NewGitHubWebhookServiceServer(&mockGitHubWebhookService{requests: pushRequests})},
	})

	// GitHub sets `created: true` on the first push of a new branch, which is
	// the main way preview deployments get triggered.
	payload := newTestPushPayload(testRepoFullName, false)
	payload.Created = true

	resp, err := sendWebhook(fmt.Sprintf("%s/webhooks/github", harness.CtrlURL), mustMarshal(t, payload), harness.Secret)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()

	select {
	case <-pushRequests:
	case <-time.After(10 * time.Second):
		t.Fatal("expected HandlePush invocation for newly created branch")
	}
}

func TestGitHubWebhook_InvalidSignature(t *testing.T) {
	pushRequests := make(chan *hydrav1.HandlePushRequest, 1)
	harness := newWebhookHarness(t, webhookHarnessConfig{
		Services: []restate.ServiceDefinition{hydrav1.NewGitHubWebhookServiceServer(&mockGitHubWebhookService{requests: pushRequests})},
	})

	resp, err := sendWebhook(fmt.Sprintf("%s/webhooks/github", harness.CtrlURL), mustMarshal(t, newTestPushPayload(testRepoFullName, false)), "wrong-secret")
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	_ = resp.Body.Close()

	select {
	case <-pushRequests:
		t.Fatal("unexpected HandlePush invocation")
	case <-time.After(1 * time.Second):
	}
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

func newTestPushPayload(repoFullName string, fork bool) pushPayload {
	prCommit := pushCommit{
		ID:        "c0",
		Message:   "feat: original PR work",
		Timestamp: "2024-01-01T00:00:00Z",
		Author:    pushCommitAuthor{Name: "pr-creator", Username: "pr-creator"},
	}
	mergeCommit := pushCommit{
		ID:        "c1",
		Message:   "Merge pull request #1 from pr-creator/feat\n\nfeat: original PR work",
		Timestamp: "2024-01-01T00:01:00Z",
		Author:    pushCommitAuthor{Name: "merger", Username: "merger"},
	}
	return pushPayload{
		Ref:          "refs/heads/main",
		After:        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Installation: pushInstallation{ID: 101},
		Repository:   pushRepository{ID: 202, FullName: repoFullName, Fork: fork},
		Commits:      []pushCommit{prCommit, mergeCommit},
		HeadCommit:   &mergeCommit,
		Sender:       pushSender{Login: "merger", AvatarURL: "https://avatar"},
	}
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
