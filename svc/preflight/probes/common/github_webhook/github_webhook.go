// Package github_webhook implements tier-1.1 of the preflight plan:
// the GitHub webhook ingress probe. It signs a synthetic push payload,
// POSTs to /webhooks/github, and asserts that a valid signature is
// accepted while a deliberately wrong signature is rejected.
//
// The reject_invalid phase is mandatory. A silently-permissive
// verifier is a security-severity regression; this probe treats a 2xx
// on the invalid-signature phase as a failure and uploads the exact
// payload for replay.
package github_webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/svc/preflight/core"
	"github.com/unkeyed/unkey/svc/preflight/probes"
)

// Probe is the tier-1.1 webhook ingress probe. See the package
// comment for the assertion surface.
type Probe struct{}

// Name implements probes.Probe.
func (Probe) Name() string { return "github_webhook" }

// Run implements probes.Probe.
func (Probe) Run(ctx context.Context, env *core.Env) core.Result {
	if env.CtrlBaseURL == "" {
		return core.Fail(errors.New("CtrlBaseURL is empty"))
	}
	if env.GitHubWebhookSecret == "" {
		return core.Fail(errors.New("GitHubWebhookSecret is empty"))
	}

	url := env.CtrlBaseURL + "/webhooks/github"
	payload, err := json.Marshal(syntheticPushPayload(env.RunID))
	if err != nil {
		return core.Failf("marshal payload: %w", err)
	}

	artifacts := []core.Artifact{artifact("request.json", payload)}
	phases := make([]core.Phase, 0, 2)

	// Phase 1: valid signature must be accepted.
	acceptStart := time.Now()
	acceptErr := sendWebhook(ctx, url, payload, env.GitHubWebhookSecret, http.StatusOK)
	phases = append(phases, core.Phase{
		Name:     "accept_valid",
		Duration: time.Since(acceptStart),
		Err:      acceptErr,
	})
	if acceptErr != nil {
		return core.Failf("valid signature rejected: %w", acceptErr).
			WithPhases(phases).
			WithArtifacts(artifacts)
	}

	// Phase 2: wrong signature MUST be rejected. A 2xx here means the
	// verifier is effectively disabled.
	rejectStart := time.Now()
	rejectErr := sendWebhook(ctx, url, payload, "wrong-secret-"+env.RunID, http.StatusUnauthorized)
	phases = append(phases, core.Phase{
		Name:     "reject_invalid",
		Duration: time.Since(rejectStart),
		Err:      rejectErr,
	})
	if rejectErr != nil {
		return core.Failf("invalid signature accepted: %w", rejectErr).
			WithPhases(phases).
			WithArtifacts(artifacts)
	}

	return core.Pass().
		WithPhases(phases).
		WithDims(map[string]string{"endpoint": url})
}

// sendWebhook posts body to url, signs it with secret, and checks the
// response status matches wantStatus. Returns nil on exact match.
func sendWebhook(ctx context.Context, url string, body []byte, secret string, wantStatus int) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-Hub-Signature-256", signWebhook(body, secret))
	req.Header.Set("X-GitHub-Delivery", "preflight-"+secret[:min(8, len(secret))])

	//nolint:exhaustruct // default transport is correct for a simple HTTP probe
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()
	// Drain body so the connection can be reused.
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != wantStatus {
		return fmt.Errorf("status %d, want %d", resp.StatusCode, wantStatus)
	}
	return nil
}

// signWebhook reproduces GitHub's HMAC-SHA256 signature format. Must
// match what svc/ctrl/api.verifyWebhookSignature expects (see
// svc/ctrl/api/github_webhook.go:67).
func signWebhook(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// syntheticPushPayload returns a valid-looking push event. Fields
// mirror the ctrl api's pushPayload struct so JSON round-trips cleanly
// through the webhook handler; unexported means we duplicate them
// here. The commit SHA is derived from runID so preflight events are
// traceable in logs.
func syntheticPushPayload(runID string) pushPayload {
	ts := time.Now().UTC().Format(time.RFC3339)
	sha := commitSHAFromRunID(runID)
	commit := pushCommit{
		ID:        sha,
		Message:   "preflight: synthetic push " + runID,
		Timestamp: ts,
		Author: pushCommitAuthor{
			Name:     "preflight-bot",
			Username: "preflight-bot",
		},
		Added:    nil,
		Removed:  nil,
		Modified: []string{".preflight-timestamp"},
	}
	return pushPayload{
		Ref:     "refs/heads/preflight-test",
		After:   sha,
		Created: false,
		Deleted: false,
		Installation: pushInstallation{
			ID: 1, // real installation id is injected via env when running against prod
		},
		Repository: pushRepository{
			ID:       1,
			FullName: "unkeyed/preflight-test-app",
			Fork:     false,
		},
		Commits:    []pushCommit{commit},
		HeadCommit: &commit,
		Sender: pushSender{
			Login:     "preflight-bot",
			AvatarURL: "https://avatars.githubusercontent.com/u/0",
		},
	}
}

func commitSHAFromRunID(runID string) string {
	h := sha256.Sum256([]byte("preflight-" + runID))
	return hex.EncodeToString(h[:20])
}

func artifact(name string, body []byte) core.Artifact {
	return core.Artifact{
		Name:        name,
		ContentType: "application/json",
		Body:        body,
	}
}

// Payload structs mirror the unexported types in svc/ctrl/api/types.go.
// They must stay in sync; a field rename on either side is caught by
// this probe's integration test.
type pushPayload struct {
	Ref          string           `json:"ref"`
	After        string           `json:"after"`
	Created      bool             `json:"created"`
	Deleted      bool             `json:"deleted"`
	Installation pushInstallation `json:"installation"`
	Repository   pushRepository   `json:"repository"`
	Commits      []pushCommit     `json:"commits"`
	HeadCommit   *pushCommit      `json:"head_commit"`
	Sender       pushSender       `json:"sender"`
}

type pushInstallation struct {
	ID int64 `json:"id"`
}

type pushRepository struct {
	ID       int64  `json:"id"`
	FullName string `json:"full_name"`
	Fork     bool   `json:"fork"`
}

type pushCommit struct {
	ID        string           `json:"id"`
	Message   string           `json:"message"`
	Timestamp string           `json:"timestamp"`
	Author    pushCommitAuthor `json:"author"`
	Added     []string         `json:"added"`
	Removed   []string         `json:"removed"`
	Modified  []string         `json:"modified"`
}

type pushCommitAuthor struct {
	Name     string `json:"name"`
	Username string `json:"username"`
}

type pushSender struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
}

func init() { probes.Register(Probe{}) }
