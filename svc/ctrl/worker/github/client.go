package github

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
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
	Message      string               `json:"message"`
	Author       ghCommitAuthor       `json:"author"`
	Verification ghCommitVerification `json:"verification"`
}

type ghCommitVerification struct {
	Verified bool `json:"verified"`
}

type ghCommitAuthor struct {
	Date  string `json:"date"`
	Name  string `json:"name"`
	Email string `json:"email"`
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
	baseURL           string
	httpClient        *http.Client
	signer            jwt.Signer[jwt.RegisteredClaims]
	tokenCache        cache.Cache[int64, InstallationToken]
	scopedTokenCache  cache.Cache[string, InstallationToken]
	collaboratorCache cache.Cache[collaboratorKey, bool]
	visibilityCache   cache.Cache[string, bool]
}

// NewClient creates a [Client] with the given configuration. Returns an error if
// the private key cannot be parsed for JWT signing.
func NewClient(config ClientConfig) (*Client, error) {
	signer, err := jwt.NewRS256Signer[jwt.RegisteredClaims](config.PrivateKeyPEM)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to create JWT signer"))
	}

	tokenCache, err := cache.New(cache.Config[int64, InstallationToken]{
		Fresh:    50 * time.Minute,
		Stale:    55 * time.Minute,
		MaxSize:  10_000,
		Resource: "github_installation_token",
		Clock:    clock.New(),
	})
	if err != nil {
		return nil, err
	}

	scopedTokenCache, err := cache.New(cache.Config[string, InstallationToken]{
		Fresh:    50 * time.Minute,
		Stale:    55 * time.Minute,
		MaxSize:  10_000,
		Resource: "github_scoped_installation_token",
		Clock:    clock.New(),
	})
	if err != nil {
		return nil, err
	}

	collaboratorCache, err := cache.New(cache.Config[collaboratorKey, bool]{
		Fresh:    5 * time.Minute,
		Stale:    30 * time.Minute,
		MaxSize:  10_000,
		Resource: "github_collaborator",
		Clock:    clock.New(),
	})
	if err != nil {
		return nil, err
	}

	// The probe is unauthenticated (60 req/hr per IP), so cache hard: repo
	// visibility almost never flips. Fresh 10m keeps a steadily-rebuilt repo to
	// ~6 probes/hr, well under the quota, and the 1h Stale window lets SWR serve
	// the cached answer while a background probe refreshes it. Either flip
	// direction fails safe for at most one cache window:
	//   - public -> private (stale "public"): the tokenless clone path 404s and
	//     the build fails closed. No access is granted, just a retry needed.
	//   - private -> public (stale "private"): a scoped read-only token is minted
	//     needlessly, but it only reads a now-public repo. Harmless.
	// A flip never escalates access; worst case it delays the optimal choice.
	visibilityCache, err := cache.New(cache.Config[string, bool]{
		Fresh:    10 * time.Minute,
		Stale:    1 * time.Hour,
		MaxSize:  10_000,
		Resource: "github_repo_visibility",
		Clock:    clock.New(),
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		config:            config,
		baseURL:           "https://api.github.com",
		httpClient:        &http.Client{Timeout: 30 * time.Second},
		signer:            signer,
		tokenCache:        tokenCache,
		scopedTokenCache:  scopedTokenCache,
		collaboratorCache: collaboratorCache,
		visibilityCache:   visibilityCache,
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

// IsRepoPublic reports whether repo ("owner/repo") is publicly accessible, using
// an unauthenticated GitHub API request. A public repo (HTTP 200) can be cloned
// by BuildKit anonymously, so a fork build needs no token at all; a private repo
// (404) requires one. This tests the exact property we care about (anonymous
// cloneability) rather than trusting a cached visibility flag.
//
// Only 200 and 404 are treated as answers. Rate limits and server errors return
// an error instead of false, so an exhausted unauthenticated quota (60 req/hr
// per IP) is never mistaken for "private". Results are cached to keep probe
// volume well under that quota.
func (c *Client) IsRepoPublic(repo string) (bool, error) {
	value, _, err := c.visibilityCache.SWR(
		context.Background(),
		repo,
		func(_ context.Context) (bool, error) {
			return probeRepoVisibility(c.httpClient, c.baseURL, repo)
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

// scopedTokenRequest is the body for POST /app/installations/{id}/access_tokens
// when minting a downscoped token. An empty body grants the App's full
// installation-wide permissions; setting these fields restricts the token.
type scopedTokenRequest struct {
	// Repositories lists repo names (not owner/repo) the token may access. When
	// omitted, the token spans all of the installation's repositories.
	Repositories []string `json:"repositories,omitempty"`
	// Permissions maps permission name to access level, e.g. {"contents":"read"}.
	// Valid names and levels:
	// https://docs.github.com/en/rest/authentication/permissions-required-for-github-apps
	Permissions map[string]string `json:"permissions"`
}

// GetScopedInstallationToken mints an installation token downscoped to the given
// permissions, e.g. {"contents":"read"}. permissions must be a subset of what
// the App was granted at install.
//
// If repo (an "owner/repo" full name) is non-empty the token is restricted to
// that single repository; if empty the token spans all of the installation's
// repositories. Callers scope to a single repo so an exfiltrated token grants
// only read access to the one repo BuildKit clones. The empty-repo (all-repos)
// form exists for callers that need cross-repo reads, but the build path does
// not use it: a single-repo read token cannot clone private cross-repo deps or
// submodules, an accepted tradeoff for keeping the build credential minimal.
//
// Results are cached (like [Client.GetInstallationToken]) keyed by the full
// scope (installation, repo, permission set) so distinct scopes never collide.
func (c *Client) GetScopedInstallationToken(installationID int64, repo string, permissions map[string]string) (InstallationToken, error) {
	if err := assert.NotNilAndNotZero(installationID, "installationID must be provided"); err != nil {
		return InstallationToken{}, err
	}

	// An empty permission set is NOT a no-op: GitHub reads it as "grant the App's
	// full installation permissions", which would silently hand back a
	// write-capable token. Reject it so the downscope is always explicit.
	if err := assert.False(len(permissions) == 0, "permissions must be provided to scope the token"); err != nil {
		return InstallationToken{}, err
	}

	// %v prints map keys in sorted order, so the key is stable across calls.
	// The permission set is part of the key: a fork's single-repo read token and
	// a trusted all-repos read token differ in scope and must never share a slot.
	key := fmt.Sprintf("%d:%s:%v", installationID, repo, permissions)

	value, _, err := c.scopedTokenCache.SWR(
		context.Background(),
		key,
		func(_ context.Context) (InstallationToken, error) {
			logger.Info(
				"Getting scoped GitHub installation token",
				"installation_id", installationID,
				"repo", repo,
			)

			var repositories []string
			if repo != "" {
				repoName := repo
				if idx := strings.LastIndex(repo, "/"); idx >= 0 {
					repoName = repo[idx+1:]
				}
				repositories = []string{repoName}
			}

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
				scopedTokenRequest{Repositories: repositories, Permissions: permissions},
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

	apiURL := fmt.Sprintf("%s/repos/%s/commits/%s", c.baseURL, repo, url.PathEscape(branch))

	commit, err := request[ghCommitResponse](c.httpClient, http.MethodGet, apiURL, headers, nil, http.StatusOK)
	if err != nil {
		logger.Error(
			"failed to fetch branch head commit",
			"installation_id", installationID,
			"repo", repo,
			"branch", branch,
			"url", apiURL,
			"err", err,
		)
		return CommitInfo{}, err
	}

	handle, avatarURL := resolveCommitAuthor(commit)
	return CommitInfoFromRaw(commit.SHA, commit.Commit.Message, handle, avatarURL, commit.Commit.Author.Date), nil
}

// GetBranchHeadCommitPublic retrieves the HEAD commit of a branch using the
// public GitHub API without authentication. Only works for public repositories.
func (c *Client) GetBranchHeadCommitPublic(repo string, branch string) (CommitInfo, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/commits/%s", c.baseURL, repo, url.PathEscape(branch))

	commit, err := request[ghCommitResponse](c.httpClient, http.MethodGet, apiURL, githubHeaders(""), nil, http.StatusOK)
	if err != nil {
		return CommitInfo{}, err
	}

	handle, avatarURL := resolveCommitAuthor(commit)
	return CommitInfoFromRaw(commit.SHA, commit.Commit.Message, handle, avatarURL, commit.Commit.Author.Date), nil
}

// GetCommitBySHA retrieves commit metadata for a specific SHA.
func (c *Client) GetCommitBySHA(installationID int64, repo string, sha string) (CommitInfo, error) {
	headers, err := c.ghHeaders(installationID)
	if err != nil {
		return CommitInfo{}, fault.Wrap(err, fault.Internal("failed to get installation token for commit lookup"))
	}

	apiURL := fmt.Sprintf("%s/repos/%s/commits/%s", c.baseURL, repo, url.PathEscape(sha))

	commit, err := request[ghCommitResponse](c.httpClient, http.MethodGet, apiURL, headers, nil, http.StatusOK)
	if err != nil {
		return CommitInfo{}, err
	}

	handle, avatarURL := resolveCommitAuthor(commit)
	return CommitInfoFromRaw(commit.SHA, commit.Commit.Message, handle, avatarURL, commit.Commit.Author.Date), nil
}

// resolveCommitAuthor picks the best author handle and avatar from a GitHub
// commit response. When the commit is verified, GitHub's resolved user is
// trustworthy. Otherwise we fall back to raw git metadata because GitHub's
// email-based resolution can map to the wrong account.
func resolveCommitAuthor(commit ghCommitResponse) (handle string, avatarURL string) {
	if commit.Commit.Verification.Verified && commit.Author.Login != "" {
		return commit.Author.Login, commit.Author.AvatarURL
	}
	name := commit.Commit.Author.Name
	email := strings.ToLower(strings.TrimSpace(commit.Commit.Author.Email))
	hash := fmt.Sprintf("%x", md5.Sum([]byte(email)))
	return name, fmt.Sprintf("https://www.gravatar.com/avatar/%s?d=identicon", hash)
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

	apiURL := fmt.Sprintf("%s/repos/%s/deployments", c.baseURL, repo)

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

	apiURL := fmt.Sprintf("%s/repos/%s/deployments/%d/statuses", c.baseURL, repo, deploymentID)

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

// CreateCommitStatus creates a commit status on a SHA using the Status API.
// Clicking "Details" in the PR goes directly to targetURL (unlike Check Runs
// which show an intermediate GitHub page).
func (c *Client) CreateCommitStatus(installationID int64, repo string, sha string, state string, targetURL string, description string, context string) error {
	headers, err := c.ghHeaders(installationID)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("%s/repos/%s/statuses/%s", c.baseURL, repo, url.PathEscape(sha))

	return doRequest(c.httpClient, http.MethodPost, apiURL, headers, map[string]string{
		"state":       state,
		"target_url":  targetURL,
		"description": description,
		"context":     context,
	}, http.StatusCreated)
}

// ghCommitFile is the subset of GitHub's commit file object that we need.
type ghCommitFile struct {
	Filename string `json:"filename"`
}

// ghCommitWithFiles extends ghCommitResponse with the files array returned by
// GET /repos/{owner}/{repo}/commits/{sha}.
type ghCommitWithFiles struct {
	Files []ghCommitFile `json:"files"`
}

// ListCommitFiles returns the list of filenames changed in a specific commit.
func (c *Client) ListCommitFiles(installationID int64, repo string, sha string) ([]string, error) {
	headers, err := c.ghHeaders(installationID)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("%s/repos/%s/commits/%s", c.baseURL, repo, url.PathEscape(sha))

	commit, err := request[ghCommitWithFiles](c.httpClient, http.MethodGet, apiURL, headers, nil, http.StatusOK)
	if err != nil {
		return nil, err
	}

	filenames := make([]string, len(commit.Files))
	for i, f := range commit.Files {
		filenames[i] = f.Filename
	}
	return filenames, nil
}

// ghPullRequest is the subset of GitHub's pull request response that we need.
type ghPullRequest struct {
	Number int `json:"number"`
}

// ghIssueComment is the subset of GitHub's issue comment response that we need.
type ghIssueComment struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
}

// FindPullRequestForBranch returns the PR number for the given branch head,
// or 0 if no open PR exists.
func (c *Client) FindPullRequestForBranch(installationID int64, repo string, branch string) (int, error) {
	headers, err := c.ghHeaders(installationID)
	if err != nil {
		return 0, err
	}

	apiURL := fmt.Sprintf("%s/repos/%s/pulls?state=open&head=%s:%s&per_page=1",
		c.baseURL, repo, strings.Split(repo, "/")[0], url.PathEscape(branch))

	prs, err := request[[]ghPullRequest](c.httpClient, http.MethodGet, apiURL, headers, nil, http.StatusOK)
	if err != nil {
		return 0, err
	}

	if len(prs) == 0 {
		return 0, nil
	}
	return prs[0].Number, nil
}

// CreateIssueComment posts a new comment on a PR/issue and returns the comment ID.
func (c *Client) CreateIssueComment(installationID int64, repo string, prNumber int, body string) (int64, error) {
	headers, err := c.ghHeaders(installationID)
	if err != nil {
		return 0, err
	}

	apiURL := fmt.Sprintf("%s/repos/%s/issues/%d/comments", c.baseURL, repo, prNumber)

	comment, err := request[ghIssueComment](c.httpClient, http.MethodPost, apiURL, headers, map[string]string{
		"body": body,
	}, http.StatusCreated)
	if err != nil {
		return 0, err
	}
	return comment.ID, nil
}

// UpdateIssueComment updates an existing PR/issue comment by ID.
func (c *Client) UpdateIssueComment(installationID int64, repo string, commentID int64, body string) error {
	headers, err := c.ghHeaders(installationID)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("%s/repos/%s/issues/comments/%d", c.baseURL, repo, commentID)

	return doRequest(c.httpClient, http.MethodPatch, apiURL, headers, map[string]string{
		"body": body,
	}, http.StatusOK)
}

// FindBotComment searches PR comments for one containing the given marker string.
// Returns the comment ID and body, or (0, "", nil) if not found.
func (c *Client) FindBotComment(installationID int64, repo string, prNumber int, marker string) (int64, string, error) {
	headers, err := c.ghHeaders(installationID)
	if err != nil {
		return 0, "", err
	}

	// Paginate through comments looking for our marker (most recent first)
	apiURL := fmt.Sprintf("%s/repos/%s/issues/%d/comments?per_page=100&direction=desc", c.baseURL, repo, prNumber)

	comments, err := request[[]ghIssueComment](c.httpClient, http.MethodGet, apiURL, headers, nil, http.StatusOK)
	if err != nil {
		return 0, "", err
	}

	for _, c := range comments {
		if strings.Contains(c.Body, marker) {
			return c.ID, c.Body, nil
		}
	}
	return 0, "", nil
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

			apiURL := fmt.Sprintf("%s/repos/%s/collaborators/%s", c.baseURL, repo, url.PathEscape(username))

			return statusCheck(c.httpClient, http.MethodGet, apiURL, headers, http.StatusNoContent)
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
