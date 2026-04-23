package github

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/runner"
)

type installationPayload struct {
	ID int64 `json:"id"`
}

type repositoryPayload struct {
	ID       int64  `json:"id"`
	FullName string `json:"full_name"`
}

type pushPayload struct {
	Ref          string              `json:"ref"`
	After        string              `json:"after"`
	Installation installationPayload `json:"installation"`
	Repository   repositoryPayload   `json:"repository"`
}

type repositoryResponse struct {
	ID            int64  `json:"id"`
	DefaultBranch string `json:"default_branch"`
}

type branchResponse struct {
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

type Service struct {
	db db.Database
}

type Config struct {
	DB db.Database
}

func New(cfg Config) (*Service, error) {
	return &Service{db: cfg.DB}, nil
}

var Cmd = &cli.Command{
	Name:  "github",
	Usage: "GitHub webhook simulation tools",
	Commands: []*cli.Command{
		triggerWebhookCmd,
		setupCmd,
		tunnelCmd,
	},
}

var triggerWebhookCmd = &cli.Command{
	Name:  "trigger-webhook",
	Usage: "Simulate a GitHub push webhook to trigger deployments locally",
	Flags: []cli.Flag{
		// Required
		cli.String("project", "Unkey project slug (e.g., 'local-api')", cli.Required()),
		cli.String("repository", "Full repository name (e.g., owner/repo)", cli.Required()),

		// Optional
		cli.String("commit-sha", "Git commit SHA to deploy; if empty, uses the HEAD of the repo's default branch"),
		cli.String("branch", "Branch name; ignored when commit-sha is empty (default branch is used)", cli.Default("main")),
		cli.String("webhook-url", "Ctrl-api webhook endpoint", cli.Default("http://localhost:7091/webhooks/github")),
		cli.String("webhook-secret", "Secret for signing; read from dev/.env.github if empty", cli.EnvVar("UNKEY_GITHUB_APP_WEBHOOK_SECRET")),
		cli.String("database-url", "MySQL connection string", cli.Default("unkey:password@tcp(127.0.0.1:3306)/unkey?parseTime=true&interpolateParams=true"), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),
	},
	Action: triggerWebhook,
}

func triggerWebhook(ctx context.Context, cmd *cli.Command) error {
	projectSlug := cmd.RequireString("project")
	repository := cmd.RequireString("repository")
	commitSHA := cmd.String("commit-sha")
	branch := cmd.String("branch")
	webhookURL := cmd.String("webhook-url")
	webhookSecret := cmd.String("webhook-secret")
	databaseURL := cmd.String("database-url")

	// Pull the secret from dev/.env.github when nothing was passed explicitly.
	// Setup wrote it there, and ctrl-api validates against that same value via its k8s secret.
	if webhookSecret == "" {
		secret, err := readEnvFileValue("dev/.env.github", "UNKEY_GITHUB_APP_WEBHOOK_SECRET")
		if err != nil {
			return fmt.Errorf("no --webhook-secret provided and failed to read dev/.env.github: %w\n\nRun `go run . dev github setup` first, or pass --webhook-secret", err)
		}
		webhookSecret = secret
	}

	r := runner.New()
	defer r.Recover()

	database, err := db.New(db.Config{
		PrimaryDSN:  databaseURL,
		ReadOnlyDSN: "",
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	r.Defer(database.Close)

	svc, err := New(Config{DB: database})
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	// Resolve project (and its workspace, which we'll need for the dashboard URL later)
	fmt.Printf("Looking up project %q...\n", projectSlug)
	project, err := db.Query.FindProjectBySlug(ctx, database.RO(), projectSlug)
	if err != nil {
		return fmt.Errorf("failed to find project with slug %q: %w\n\nMake sure you've run `go run . dev seed local`", projectSlug, err)
	}
	workspace, err := db.Query.FindWorkspaceByID(ctx, database.RO(), project.WorkspaceID)
	if err != nil {
		return fmt.Errorf("failed to load workspace %q for project %q: %w", project.WorkspaceID, projectSlug, err)
	}
	projectID := project.ID
	fmt.Printf("Project: %s (id: %s)\n", project.Slug, projectID)

	// Fetch repo info from GitHub API
	fmt.Printf("Fetching repository info for %s...\n", repository)
	repoInfo, err := fetchRepositoryInfo(ctx, repository)
	if err != nil {
		return fmt.Errorf("failed to fetch repository: %w\n\nMake sure the repository exists and is publicly accessible", err)
	}
	fmt.Printf("Repository: %s (id: %d)\n", repository, repoInfo.ID)

	// When no SHA is given, deploy HEAD of the repo's default branch
	if commitSHA == "" {
		branch = repoInfo.DefaultBranch
		fmt.Printf("No --commit-sha provided; resolving HEAD of default branch %q...\n", branch)
		commitSHA, err = fetchBranchHeadSHA(ctx, repository, branch)
		if err != nil {
			return fmt.Errorf("failed to resolve HEAD SHA for branch %q: %w", branch, err)
		}
	}

	// Resolve the real GitHub App installation ID. This is what the build
	// path uses to mint a GitHub token and clone the repo, exactly like a
	// production push event.
	installationID, err := resolveInstallationID(ctx, repository)
	if err != nil {
		return fmt.Errorf("failed to resolve GitHub App installation: %w", err)
	}
	fmt.Printf("Installation: %d\n", installationID)

	// Resolve the default app for this project
	fmt.Println("Looking up default app for project...")
	appRow, err := db.Query.FindAppByProjectAndSlug(ctx, database.RO(), db.FindAppByProjectAndSlugParams{
		ProjectID: projectID,
		Slug:      "default",
	})
	if err != nil {
		return fmt.Errorf("failed to find default app for project %s: %w\n\nMake sure the project has a 'default' app", projectID, err)
	}
	fmt.Printf("App: %s (id: %s)\n", appRow.App.Slug, appRow.App.ID)

	// Ensure github_repo_connection exists
	fmt.Println("Ensuring GitHub connection exists in database...")
	if err := svc.ensureGithubConnection(ctx, appRow.App.WorkspaceID, projectID, appRow.App.ID, installationID, repoInfo.ID, repository); err != nil {
		return fmt.Errorf("failed to create GitHub connection: %w", err)
	}
	fmt.Println("✔ GitHub connection ready")

	payload := pushPayload{
		Ref:   fmt.Sprintf("refs/heads/%s", branch),
		After: commitSHA,
		Installation: installationPayload{
			ID: installationID,
		},
		Repository: repositoryPayload{
			ID:       repoInfo.ID,
			FullName: repository,
		},
	}

	// Serialize to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to serialize payload: %w", err)
	}

	// Generate signature
	signature := generateSignature(payloadBytes, webhookSecret)

	// Send request
	fmt.Println("\nTriggering deployment webhook...")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-Hub-Signature-256", signature)
	req.Header.Set("User-Agent", "unkey-cli-webhook-trigger")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("✗ Failed to connect to webhook endpoint: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle response
	switch resp.StatusCode {
	case http.StatusOK:
		fmt.Println("✔ Webhook delivered successfully")
		fmt.Println("✔ Deployment workflow started")
		fmt.Println()
		fmt.Printf("Repository: %s\n", repository)
		fmt.Printf("Branch: %s\n", branch)
		fmt.Printf("Commit: %s\n", commitSHA[:min(8, len(commitSHA))])

		// Determine environment
		env := "preview"
		if branch == "main" || branch == "master" {
			env = "production"
		}
		fmt.Printf("Environment: %s\n", env)
		fmt.Printf("Dashboard: http://localhost:3000/%s/projects/%s/deployments\n", workspace.Slug, projectID)
		return nil

	case http.StatusUnauthorized:
		return fmt.Errorf("✗ Webhook rejected: invalid signature\n\nThe webhook secret used to sign this request does not match the one ctrl-api is validating against. Make sure dev/.env.github (UNKEY_GITHUB_APP_WEBHOOK_SECRET) is the one written by `go run . dev github setup` and that the github-credentials k8s secret is up to date")

	case http.StatusBadRequest:
		return fmt.Errorf("✗ Webhook rejected: invalid payload\n\n%s", string(bodyBytes))

	case http.StatusInternalServerError:
		return fmt.Errorf("✗ Server error: %s\n\n%s", resp.Status, string(bodyBytes))

	default:
		return fmt.Errorf("✗ Unexpected response: %s\n\n%s", resp.Status, string(bodyBytes))
	}
}

func (s *Service) ensureGithubConnection(ctx context.Context, workspaceID, projectID, appID string, installationID, repositoryID int64, repository string) error {
	// Try to insert, ignore if already exists
	err := db.Query.InsertGithubRepoConnection(ctx, s.db.RW(), db.InsertGithubRepoConnectionParams{
		WorkspaceID:        workspaceID,
		ProjectID:          projectID,
		AppID:              appID,
		InstallationID:     installationID,
		RepositoryID:       repositoryID,
		RepositoryFullName: repository,
		CreatedAt:          time.Now().UnixMilli(),
		UpdatedAt:          sql.NullInt64{Valid: false, Int64: 0},
	})
	if err != nil && !db.IsDuplicateKeyError(err) {
		return fmt.Errorf("failed to insert connection: %w", err)
	}

	return nil
}

// fetchRepositoryInfo returns the repo's numeric ID and default branch via the
// public /repos/{owner}/{repo} endpoint. No auth is used, so this works only
// for public repositories.
func fetchRepositoryInfo(ctx context.Context, repository string) (repositoryResponse, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s", repository)
	body, err := githubGET(ctx, url, repository)
	if err != nil {
		return repositoryResponse{}, err
	}

	var result repositoryResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return repositoryResponse{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// fetchBranchHeadSHA returns the HEAD commit SHA of a branch via the public
// /repos/{owner}/{repo}/branches/{branch} endpoint. Used to resolve a
// deployable SHA when the caller doesn't pin one.
func fetchBranchHeadSHA(ctx context.Context, repository, branch string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/branches/%s", repository, branch)
	body, err := githubGET(ctx, url, repository)
	if err != nil {
		return "", err
	}

	var result branchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}
	if result.Commit.SHA == "" {
		return "", fmt.Errorf("branch %q has no commit SHA", branch)
	}
	return result.Commit.SHA, nil
}

// githubGET performs an unauthenticated GET against the GitHub public REST API.
// The repository argument is only used to produce a more readable error message
// on 404/401; it's not part of the request.
func githubGET(ctx context.Context, url, repository string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "unkey-cli")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return body, nil
	case http.StatusNotFound:
		return nil, fmt.Errorf("not found: %s\n\nThis tool only works with public repositories. If '%s' is private, make it public", url, repository)
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, fmt.Errorf("authentication required: %s\n\nThis tool only works with public repositories", url)
	default:
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}
}

// resolveInstallationID asks GitHub which installation of our dev App (if any)
// is installed on the target repository, so the build path can mint a real
// installation token just like a production push event. When the App is not
// yet installed it prints the install URL and polls until the user clicks
// Install in their browser.
func resolveInstallationID(ctx context.Context, repository string) (int64, error) {
	appID, err := readAppID("dev/.env.github")
	if err != nil {
		return 0, err
	}
	pem, err := os.ReadFile(filepath.Clean("dev/.github-private-key.pem"))
	if err != nil {
		return 0, fmt.Errorf("failed to read dev/.github-private-key.pem: %w", err)
	}
	jwtToken, err := generateAppJWT(appID, string(pem))
	if err != nil {
		return 0, fmt.Errorf("failed to sign GitHub App JWT: %w", err)
	}

	id, found, err := fetchInstallationID(ctx, repository, jwtToken)
	if err != nil {
		return 0, err
	}
	if found {
		return id, nil
	}

	appSlug, _ := readEnvFileValue("dev/.env.github", "NEXT_PUBLIC_GITHUB_APP_NAME")
	fmt.Printf("\nThe dev GitHub App is not installed on %s.\n", repository)
	fmt.Printf("Install it here, then this command will continue automatically:\n\n  https://github.com/apps/%s/installations/new\n\n", appSlug)
	fmt.Print("Waiting for installation")

	// App JWTs live 10 minutes; cap the poll below that.
	pollCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-pollCtx.Done():
			fmt.Println()
			if errors.Is(pollCtx.Err(), context.DeadlineExceeded) {
				return 0, fmt.Errorf("timed out waiting for GitHub App installation on %s", repository)
			}
			return 0, pollCtx.Err()
		case <-ticker.C:
			fmt.Print(".")
			id, found, err := fetchInstallationID(ctx, repository, jwtToken)
			if err != nil {
				fmt.Println()
				return 0, err
			}
			if found {
				fmt.Println(" ✔")
				return id, nil
			}
		}
	}
}

// fetchInstallationID does a single GET /repos/{repo}/installation. The bool
// return is false (with a nil error) when GitHub responds 404, meaning the App
// is not installed on the repo yet. That's the normal "keep polling" signal.
func fetchInstallationID(ctx context.Context, repository, jwtToken string) (int64, bool, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/installation", repository)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, false, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "unkey-cli")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return 0, false, fmt.Errorf("GitHub request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, false, fmt.Errorf("failed to read response: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var result struct {
			ID int64 `json:"id"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return 0, false, fmt.Errorf("failed to parse installation response: %w", err)
		}
		return result.ID, true, nil
	case http.StatusNotFound:
		return 0, false, nil
	default:
		return 0, false, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}
}
