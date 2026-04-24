// Package githubpush is preflight's minimal GitHub App writer. It
// authenticates as a configured App, swaps the App JWT for an
// installation token, and creates or updates a single file on a
// branch via GitHub's Contents API. Everything else the preflight
// tier-1.10 git_push probe needs from GitHub goes through here.
//
// We do not import svc/ctrl/worker/github: that client has no
// commit-write primitives (only reads + deployment-status writes) and
// lives inside ctrl's worker, which preflight has no business
// depending on. The surface here is narrow enough that hand-rolling it
// is clearer than pulling in a general-purpose GitHub SDK.
package githubpush

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/jwt"
)

// HTTPClient is the single method githubpush needs from *http.Client.
// Accepting the interface lets tests drive the client against an
// httptest.Server without exposing the real transport.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Config holds the App credentials plus everything the client needs
// to rotate installation tokens.
type Config struct {
	AppID          int64
	InstallationID int64
	PrivateKeyPEM  string

	// BaseURL is the GitHub API root. Defaults to
	// https://api.github.com when empty; tests override to point at a
	// local httptest.Server.
	BaseURL string

	// HTTPClient is the transport used for all calls. Defaults to
	// http.DefaultClient when nil; tests substitute a recording client.
	HTTPClient HTTPClient

	// Now returns the current time. Defaults to time.Now; tests pin to
	// a deterministic clock when asserting on the App JWT iat/exp.
	Now func() time.Time
}

// Client wraps a GitHub App + installation so callers issue a single
// PushFile call and get back the new commit SHA. The installation
// token is cached and refreshed lazily five minutes before expiry.
type Client struct {
	cfg       Config
	signer    jwt.Signer[jwt.RegisteredClaims]
	tokMu     sync.Mutex
	tokCache  string
	tokExpiry time.Time
}

// New constructs a Client from cfg. Returns an error if the private
// key cannot be parsed.
func New(cfg Config) (*Client, error) {
	if cfg.AppID == 0 {
		return nil, fmt.Errorf("githubpush: AppID is zero")
	}
	if cfg.InstallationID == 0 {
		return nil, fmt.Errorf("githubpush: InstallationID is zero")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.github.com"
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	signer, err := jwt.NewRS256Signer[jwt.RegisteredClaims](cfg.PrivateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("githubpush: parse private key: %w", err)
	}
	return &Client{
		cfg:       cfg,
		signer:    signer,
		tokMu:     sync.Mutex{},
		tokCache:  "",
		tokExpiry: time.Time{},
	}, nil
}

// PushFile creates or updates `path` on `branch` of `repo` with the
// given `content` and commit `message`. Returns the new commit SHA on
// success. Safe to call for a path that does not yet exist (GitHub's
// Contents API handles create-or-update uniformly when the caller
// omits the existing sha on new files and supplies it on updates).
//
// `repo` is "owner/repo".
func (c *Client) PushFile(ctx context.Context, repo, branch, path string, content []byte, message string) (string, error) {
	token, err := c.installationToken(ctx)
	if err != nil {
		return "", err
	}

	if err := c.ensureBranch(ctx, token, repo, branch); err != nil {
		return "", fmt.Errorf("ensure branch: %w", err)
	}

	existingSHA, err := c.lookupFileSHA(ctx, token, repo, branch, path)
	if err != nil {
		return "", fmt.Errorf("lookup existing file: %w", err)
	}

	payload := struct {
		Message string `json:"message"`
		Content string `json:"content"`
		Branch  string `json:"branch"`
		SHA     string `json:"sha,omitempty"`
	}{
		Message: message,
		Content: base64.StdEncoding.EncodeToString(content),
		Branch:  branch,
		SHA:     existingSHA,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/repos/%s/contents/%s", c.cfg.BaseURL, repo, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("put contents: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("contents API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var result struct {
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if result.Commit.SHA == "" {
		return "", fmt.Errorf("contents API returned empty commit sha")
	}
	return result.Commit.SHA, nil
}

// installationToken returns a valid installation access token,
// rotating five minutes before the cached one expires.
func (c *Client) installationToken(ctx context.Context) (string, error) {
	c.tokMu.Lock()
	defer c.tokMu.Unlock()

	now := c.cfg.Now()
	if c.tokCache != "" && c.tokExpiry.After(now.Add(5*time.Minute)) {
		return c.tokCache, nil
	}

	appJWT, err := c.signAppJWT(now)
	if err != nil {
		return "", fmt.Errorf("sign app JWT: %w", err)
	}

	url := fmt.Sprintf("%s/app/installations/%d/access_tokens", c.cfg.BaseURL, c.cfg.InstallationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+appJWT)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("exchange installation token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("installation token exchange: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if result.Token == "" {
		return "", fmt.Errorf("installation token exchange returned empty token")
	}

	c.tokCache = result.Token
	c.tokExpiry = result.ExpiresAt
	return c.tokCache, nil
}

// signAppJWT mints the short-lived JWT GitHub accepts to exchange for
// an installation token. iat is set 60s in the past to tolerate small
// clock skew; exp is 9 minutes to stay under GitHub's 10-minute cap.
func (c *Client) signAppJWT(now time.Time) (string, error) {
	claims := jwt.RegisteredClaims{
		Issuer:    fmt.Sprintf("%d", c.cfg.AppID),
		IssuedAt:  now.Add(-60 * time.Second).Unix(),
		ExpiresAt: now.Add(9 * time.Minute).Unix(),
		Subject:   "",
		Audience:  nil,
		NotBefore: 0,
		ID:        "",
	}
	return c.signer.Sign(claims)
}

// ensureBranch creates branch from the repo's default branch when it
// does not exist. The Contents API does not auto-create branches, so
// without this PushFile against a fresh per-region branch fails with
// "Branch X not found" on the very first run.
func (c *Client) ensureBranch(ctx context.Context, token, repo, branch string) error {
	exists, err := c.branchExists(ctx, token, repo, branch)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	defaultBranch, err := c.defaultBranch(ctx, token, repo)
	if err != nil {
		return fmt.Errorf("look up default branch: %w", err)
	}
	if defaultBranch == branch {
		return fmt.Errorf("default branch %q does not exist on %s; repo appears empty", branch, repo)
	}

	headSHA, err := c.refSHA(ctx, token, repo, "heads/"+defaultBranch)
	if err != nil {
		return fmt.Errorf("resolve %s SHA: %w", defaultBranch, err)
	}

	return c.createRef(ctx, token, repo, "refs/heads/"+branch, headSHA)
}

func (c *Client) branchExists(ctx context.Context, token, repo, branch string) (bool, error) {
	url := fmt.Sprintf("%s/repos/%s/branches/%s", c.cfg.BaseURL, repo, branch)
	resp, err := c.ghGET(ctx, token, url)
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()
	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("branches GET: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
}

func (c *Client) defaultBranch(ctx context.Context, token, repo string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s", c.cfg.BaseURL, repo)
	resp, err := c.ghGET(ctx, token, url)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("repo GET: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var result struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode repo GET: %w", err)
	}
	if result.DefaultBranch == "" {
		return "", fmt.Errorf("repo GET returned empty default_branch")
	}
	return result.DefaultBranch, nil
}

func (c *Client) refSHA(ctx context.Context, token, repo, ref string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/git/ref/%s", c.cfg.BaseURL, repo, ref)
	resp, err := c.ghGET(ctx, token, url)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ref GET: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var result struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode ref GET: %w", err)
	}
	if result.Object.SHA == "" {
		return "", fmt.Errorf("ref GET returned empty sha")
	}
	return result.Object.SHA, nil
}

func (c *Client) createRef(ctx context.Context, token, repo, ref, sha string) error {
	body, err := json.Marshal(struct {
		Ref string `json:"ref"`
		SHA string `json:"sha"`
	}{Ref: ref, SHA: sha})
	if err != nil {
		return fmt.Errorf("marshal ref payload: %w", err)
	}

	url := fmt.Sprintf("%s/repos/%s/git/refs", c.cfg.BaseURL, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("create ref: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create ref: %d %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}

// ghGET is the small boilerplate-eraser shared by the read-only
// helpers above. Tests still substitute via cfg.HTTPClient.
func (c *Client) ghGET(ctx context.Context, token, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	return c.cfg.HTTPClient.Do(req)
}

// lookupFileSHA returns the existing file's SHA when the file is
// present on the branch, or an empty string when it does not exist.
// A non-404 error other than "not found" is returned verbatim so
// PushFile can distinguish "file absent" from "GitHub is down".
func (c *Client) lookupFileSHA(ctx context.Context, token, repo, branch, path string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", c.cfg.BaseURL, repo, path, branch)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("contents GET: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var result struct {
		SHA string `json:"sha"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode contents GET: %w", err)
	}
	return result.SHA, nil
}
