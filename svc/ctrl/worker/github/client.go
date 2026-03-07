package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/pkg/logger"
)

// ghCommitResponse is the subset of GitHub's GET /repos/{owner}/{repo}/commits/{ref}
// response that we need.
type ghCommitResponse struct {
	SHA    string         `json:"sha"`
	Commit ghCommitDetail `json:"commit"`
	Author ghUser         `json:"author"`
}

type ghCommitDetail struct {
	Message string         `json:"message"`
	Author  ghCommitAuthor `json:"author"`
}

type ghCommitAuthor struct {
	Date string `json:"date"`
}

type ghUser struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
}

// ghDeploymentResponse is the subset of GitHub's POST /repos/{owner}/{repo}/deployments
// response that we need.
type ghDeploymentResponse struct {
	ID int64 `json:"id"`
}

// ClientConfig holds configuration for creating a [Client] instance.
type ClientConfig struct {
	// AppID is the numeric ID assigned to the GitHub App during registration.
	AppID int64

	// PrivateKeyPEM is the RSA private key in PEM format for signing JWTs.
	// Generate this in the GitHub App settings under "Private keys".
	PrivateKeyPEM string

	// WebhookSecret is the shared secret for verifying webhook signatures.
	// Set this in the GitHub App settings under "Webhook secret".
	WebhookSecret string
}

// Ensure Client implements GitHubClient
var _ GitHubClient = (*Client)(nil)

// collaboratorKey uniquely identifies a user+repo pair for cache lookups.
type collaboratorKey struct {
	installationID int64
	repo           string
	username       string
}

// Client provides access to GitHub API using App authentication.
//
// Client handles JWT generation for App-level authentication and installation
// token retrieval for repository-level operations. It is safe for concurrent use.
type Client struct {
	config            ClientConfig
	httpClient        *http.Client
	signer            jwt.Signer[jwt.RegisteredClaims]
	tokenCache        cache.Cache[int64, InstallationToken]
	collaboratorCache cache.Cache[collaboratorKey, bool]
}

// NewClient creates a [Client] with the given configuration. Returns an error if
// the private key cannot be parsed for JWT signing.
func NewClient(config ClientConfig) (*Client, error) {
	signer, err := jwt.NewRS256Signer[jwt.RegisteredClaims](config.PrivateKeyPEM)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to create JWT signer"))
	}

	tokenCache, err := cache.New(cache.Config[int64, InstallationToken]{
		Fresh:    55 * time.Minute,
		Stale:    5 * time.Minute,
		MaxSize:  10_000,
		Resource: "github_installation_token",
		Clock:    clock.New(),
	})
	if err != nil {
		return nil, err
	}

	collaboratorCache, err := cache.New(cache.Config[collaboratorKey, bool]{
		Fresh:    5 * time.Minute,
		Stale:    1 * time.Minute,
		MaxSize:  10_000,
		Resource: "github_collaborator",
		Clock:    clock.New(),
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		config:            config,
		httpClient:        &http.Client{Timeout: 30 * time.Second},
		signer:            signer,
		tokenCache:        tokenCache,
		collaboratorCache: collaboratorCache,
	}, nil
}

// ghHeaders returns GitHub API headers authenticated with the given installation.
func (c *Client) ghHeaders(installationID int64) (map[string]string, error) {
	token, err := c.GetInstallationToken(installationID)
	if err != nil {
		return nil, err
	}
	return githubHeaders(token.Token), nil
}

// generateJWT creates a short-lived JWT for GitHub App authentication.
func (c *Client) generateJWT() (string, error) {
	now := time.Now()
	// nolint:exhaustruct
	claims := jwt.RegisteredClaims{
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(10 * time.Minute).Unix(),
		Issuer:    fmt.Sprintf("%d", c.config.AppID),
	}
	return c.signer.Sign(claims)
}

// GetInstallationToken retrieves an access token for a specific installation.
func (c *Client) GetInstallationToken(installationID int64) (InstallationToken, error) {
	if err := assert.NotNilAndNotZero(installationID, "installationID must be provided"); err != nil {
		return InstallationToken{}, err
	}

	value, _, err := c.tokenCache.SWR(
		context.Background(),
		installationID,
		func(_ context.Context) (InstallationToken, error) {
			logger.Info(
				"Getting GitHub installation token",
				"installation_id", installationID,
			)

			jwtToken, err := c.generateJWT()
			if err != nil {
				return InstallationToken{}, err
			}

			apiURL := fmt.Sprintf(
				"https://api.github.com/app/installations/%d/access_tokens",
				installationID,
			)

			return request[InstallationToken](
				c.httpClient,
				http.MethodPost,
				apiURL,
				githubHeaders(jwtToken),
				nil,
				http.StatusCreated,
			)
		},
		func(err error) cache.Op {
			if err != nil {
				return cache.Noop
			}
			return cache.WriteValue
		},
	)
	if err != nil {
		return InstallationToken{}, err
	}

	return value, nil
}

// GetBranchHeadCommit retrieves the HEAD commit of a branch from a GitHub
// repository. It uses an installation token to authenticate the request.
func (c *Client) GetBranchHeadCommit(installationID int64, repo string, branch string) (CommitInfo, error) {
	headers, err := c.ghHeaders(installationID)
	if err != nil {
		return CommitInfo{}, fault.Wrap(err, fault.Internal("failed to get installation token for branch head lookup"))
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/commits/%s", repo, url.PathEscape(branch))

	commit, err := request[ghCommitResponse](c.httpClient, http.MethodGet, apiURL, headers, nil, http.StatusOK)
	if err != nil {
		return CommitInfo{}, err
	}

	return CommitInfoFromRaw(
		commit.SHA,
		commit.Commit.Message,
		commit.Author.Login,
		commit.Author.AvatarURL,
		commit.Commit.Author.Date,
	), nil
}

// GetBranchHeadCommitPublic retrieves the HEAD commit of a branch using the
// public GitHub API without authentication. Only works for public repositories.
func (c *Client) GetBranchHeadCommitPublic(repo string, branch string) (CommitInfo, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/commits/%s", repo, url.PathEscape(branch))

	commit, err := request[ghCommitResponse](c.httpClient, http.MethodGet, apiURL, githubHeaders(""), nil, http.StatusOK)
	if err != nil {
		return CommitInfo{}, err
	}

	return CommitInfoFromRaw(
		commit.SHA,
		commit.Commit.Message,
		commit.Author.Login,
		commit.Author.AvatarURL,
		commit.Commit.Author.Date,
	), nil
}

// CommitInfoFromRaw constructs a CommitInfo, truncating the message to the
// first line and parsing an RFC3339 timestamp string.
func CommitInfoFromRaw(sha, message, authorHandle, authorAvatarURL, timestamp string) CommitInfo {
	if idx := strings.Index(message, "\n"); idx != -1 {
		message = message[:idx]
	}

	var ts time.Time
	if timestamp != "" {
		if parsed, err := time.Parse(time.RFC3339, timestamp); err == nil {
			ts = parsed
		}
	}

	return CommitInfo{
		SHA:             sha,
		Message:         message,
		AuthorHandle:    authorHandle,
		AuthorAvatarURL: authorAvatarURL,
		Timestamp:       ts,
	}
}

// VerifyWebhookSignature verifies a GitHub webhook signature using constant-time
// comparison.
func VerifyWebhookSignature(payload []byte, signature, secret string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	expectedSig := signature[7:]

	mac := hmacSHA256([]byte(secret), payload)
	actualSig := fmt.Sprintf("%x", mac)

	return hmacEqual([]byte(expectedSig), []byte(actualSig))
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func hmacEqual(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// CreateDeployment creates a GitHub Deployment on a commit ref and returns the
// deployment ID. The deployment appears in the PR sidebar on GitHub.
func (c *Client) CreateDeployment(installationID int64, repo string, ref string, environment string, description string, isProduction bool) (int64, error) {
	headers, err := c.ghHeaders(installationID)
	if err != nil {
		return 0, err
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/deployments", repo)

	result, err := request[ghDeploymentResponse](c.httpClient, http.MethodPost, apiURL, headers, map[string]interface{}{
		"ref":                    ref,
		"environment":            environment,
		"description":            description,
		"auto_merge":             false,
		"required_contexts":      []string{},
		"transient_environment":  !isProduction,
		"production_environment": isProduction,
	}, http.StatusCreated)
	if err != nil {
		return 0, err
	}

	return result.ID, nil
}

// CreateDeploymentStatus updates the status of a GitHub Deployment.
func (c *Client) CreateDeploymentStatus(installationID int64, repo string, deploymentID int64, state string, environmentURL string, logURL string, description string) error {
	headers, err := c.ghHeaders(installationID)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/deployments/%d/statuses", repo, deploymentID)

	payload := map[string]interface{}{
		"state":         state,
		"description":   description,
		"auto_inactive": true,
	}
	if environmentURL != "" {
		payload["environment_url"] = environmentURL
	}
	if logURL != "" {
		payload["log_url"] = logURL
	}

	return doRequest(c.httpClient, http.MethodPost, apiURL, headers, payload, http.StatusCreated)
}

// IsCollaborator checks whether a GitHub user is a collaborator on a repository.
// Results are cached for 5 minutes to avoid redundant API calls for the same user.
func (c *Client) IsCollaborator(installationID int64, repo string, username string) (bool, error) {
	key := collaboratorKey{installationID: installationID, repo: repo, username: username}

	value, _, err := c.collaboratorCache.SWR(
		context.Background(),
		key,
		func(_ context.Context) (bool, error) {
			headers, err := c.ghHeaders(installationID)
			if err != nil {
				return false, err
			}

			apiURL := fmt.Sprintf("https://api.github.com/repos/%s/collaborators/%s", repo, url.PathEscape(username))

			return httpclient.StatusCheck(c.httpClient, http.MethodGet, apiURL, headers, http.StatusNoContent)
		},
		func(err error) cache.Op {
			if err != nil {
				return cache.Noop
			}
			return cache.WriteValue
		},
	)
	if err != nil {
		return false, err
	}

	return value, nil
}

// FindPullRequestForBranch finds an open pull request for the given branch.
// Returns the PR number, or 0 if no open PR exists.
func (c *Client) FindPullRequestForBranch(installationID int64, repo string, branch string) (int, error) {
	headers, err := c.ghHeaders(installationID)
	if err != nil {
		return 0, err
	}

	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return 0, fault.New("invalid repo format, expected owner/repo")
	}
	owner := parts[0]

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/pulls?head=%s:%s&state=open&per_page=1",
		repo, url.QueryEscape(owner), url.QueryEscape(branch))

	type ghPR struct {
		Number int `json:"number"`
	}

	prs, err := httpclient.Request[[]ghPR](c.httpClient, http.MethodGet, apiURL, headers, nil, http.StatusOK)
	if err != nil {
		return 0, err
	}

	if len(prs) == 0 {
		return 0, nil
	}

	return prs[0].Number, nil
}

// CreateIssueComment posts a comment on a GitHub issue or pull request.
func (c *Client) CreateIssueComment(installationID int64, repo string, issueNumber int, body string) error {
	headers, err := c.ghHeaders(installationID)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d/comments", repo, issueNumber)

	return httpclient.Do(c.httpClient, http.MethodPost, apiURL, headers, map[string]string{"body": body}, http.StatusCreated)
}

// UpdateIssueComment updates an existing comment on a GitHub issue or PR.
func (c *Client) UpdateIssueComment(installationID int64, repo string, commentID int64, body string) error {
	headers, err := c.ghHeaders(installationID)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/issues/comments/%d", repo, commentID)

	return httpclient.Do(c.httpClient, http.MethodPatch, apiURL, headers, map[string]string{"body": body}, http.StatusOK)
}

// FindBotComment searches for a comment on a PR that contains the given marker
// string. Returns the comment ID if found, or 0 if no matching comment exists.
// Paginates through all comments to find it.
func (c *Client) FindBotComment(installationID int64, repo string, issueNumber int, marker string) (int64, error) {
	headers, err := c.ghHeaders(installationID)
	if err != nil {
		return 0, err
	}

	type ghComment struct {
		ID   int64  `json:"id"`
		Body string `json:"body"`
	}

	page := 1
	for {
		apiURL := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d/comments?per_page=100&page=%d", repo, issueNumber, page)

		comments, reqErr := httpclient.Request[[]ghComment](c.httpClient, http.MethodGet, apiURL, headers, nil, http.StatusOK)
		if reqErr != nil {
			return 0, reqErr
		}

		for _, comment := range comments {
			if strings.Contains(comment.Body, marker) {
				return comment.ID, nil
			}
		}

		if len(comments) < 100 {
			break
		}
		page++
	}

	return 0, nil
}
