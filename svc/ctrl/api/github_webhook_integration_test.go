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
	"github.com/unkeyed/unkey/pkg/uid"
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
	payload["deleted"] = true
	payload["after"] = "0000000000000000000000000000000000000000"

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
	payload["created"] = true

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
	// Unique per request: the handler forwards this as the Restate idempotency
	// key, so a constant would make every push after the first dedupe to the
	// first invocation and the handler would never run again.
	req.Header.Set("X-GitHub-Delivery", uid.New("delivery"))
	req.Header.Set("X-Hub-Signature-256", sign(body, secret))

	client := &http.Client{Timeout: 10 * time.Second}
	return client.Do(req)
}

// newTestPushPayload builds the push webhook body as a raw JSON map so this
// test stays decoupled from the handler's (now package-private) payload types.
func newTestPushPayload(repoFullName string, fork bool) map[string]any {
	commit := func(id, msg, ts, name string) map[string]any {
		return map[string]any{
			"id":        id,
			"message":   msg,
			"timestamp": ts,
			"author":    map[string]any{"name": name, "username": name},
		}
	}
	merge := commit("c1", "Merge pull request #1 from pr-creator/feat\n\nfeat: original PR work", "2024-01-01T00:01:00Z", "merger")
	return map[string]any{
		"ref":          "refs/heads/main",
		"after":        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"installation": map[string]any{"id": 101},
		"repository":   map[string]any{"id": 202, "full_name": repoFullName, "fork": fork},
		"commits": []map[string]any{
			commit("c0", "feat: original PR work", "2024-01-01T00:00:00Z", "pr-creator"),
			merge,
		},
		"head_commit": merge,
		"sender":      map[string]any{"login": "merger", "avatar_url": "https://avatar"},
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
