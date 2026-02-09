package github

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/db"
)

type installationPayload struct {
	ID int64 `json:"id"`
}

type repositoryPayload struct {
	ID       int64  `json:"id"`
	FullName string `json:"full_name"`
}

type pushPayload struct {
	Ref          string               `json:"ref"`
	After        string               `json:"after"`
	Installation installationPayload `json:"installation"`
	Repository   repositoryPayload   `json:"repository"`
}

var Cmd = &cli.Command{
	Name:  "github",
	Usage: "GitHub webhook simulation tools",
	Commands: []*cli.Command{
		triggerWebhookCmd,
	},
}

var triggerWebhookCmd = &cli.Command{
	Name:  "trigger-webhook",
	Usage: "Simulate a GitHub push webhook to trigger deployments locally",
	Flags: []cli.Flag{
		// Required
		cli.String("project-id", "Unkey project ID", cli.Required()),
		cli.String("repository", "Full repository name (e.g., owner/repo)", cli.Required()),
		cli.String("commit-sha", "Git commit SHA to deploy", cli.Required()),

		// Optional
		cli.String("branch", "Branch name", cli.Default("main")),
		cli.String("webhook-url", "Ctrl-api webhook endpoint", cli.Default("http://localhost:7091/webhooks/github")),
		cli.String("webhook-secret", "Secret for signing", cli.Default("supersecret"), cli.EnvVar("UNKEY_GITHUB_APP_WEBHOOK_SECRET")),
		cli.String("database-url", "MySQL connection string", cli.Default("unkey:password@tcp(127.0.0.1:3306)/unkey?parseTime=true&interpolateParams=true"), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),
	},
	Action: triggerWebhook,
}

func triggerWebhook(ctx context.Context, cmd *cli.Command) error {
	projectID := cmd.RequireString("project-id")
	repository := cmd.RequireString("repository")
	commitSHA := cmd.RequireString("commit-sha")
	branch := cmd.String("branch")
	webhookURL := cmd.String("webhook-url")
	webhookSecret := cmd.String("webhook-secret")
	databaseURL := cmd.String("database-url")

	// Fetch repository ID from GitHub API
	fmt.Printf("Fetching repository ID for %s...\n", repository)
	repositoryID, err := fetchRepositoryID(ctx, repository)
	if err != nil {
		return fmt.Errorf("failed to fetch repository ID: %w\n\nMake sure the repository exists and is publicly accessible", err)
	}

	// Use dummy installation ID for local dev
	installationID := int64(1)

	fmt.Printf("Repository: %s (id: %d)\n", repository, repositoryID)
	fmt.Printf("Project: %s\n", projectID)

	// Ensure github_repo_connection exists
	fmt.Println("Ensuring GitHub connection exists in database...")
	if err := ensureGithubConnection(ctx, databaseURL, projectID, installationID, repositoryID, repository); err != nil {
		return fmt.Errorf("failed to create GitHub connection: %w", err)
	}
	fmt.Println("✔ GitHub connection ready")

	// Warn if deployment will fail due to auth configuration
	if !checkAllowUnauthenticatedDeployments() && installationID == 1 {
		fmt.Println()
		fmt.Println("   WARNING: Deployment will fail during build phase")
		fmt.Println("   Reason: Using dummy installation ID (1) requires UNKEY_ALLOW_UNAUTHENTICATED_DEPLOYMENTS=true")
		fmt.Println()
		fmt.Println("   Fix: Add to dev/.env.github:")
		fmt.Println("   UNKEY_ALLOW_UNAUTHENTICATED_DEPLOYMENTS=true")
		fmt.Println()
		fmt.Println("   NOTE: Builds also require depot credentials in dev/.env.depot:")
		fmt.Println("   DEPOT_TOKEN=your_depot_token_here")
		fmt.Println()
		os.Exit(0)
	}

	payload := pushPayload{
		Ref:   fmt.Sprintf("refs/heads/%s", branch),
		After: commitSHA,
		Installation: installationPayload{
			ID: installationID,
		},
		Repository: repositoryPayload{
			ID:       repositoryID,
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
		return nil

	case http.StatusUnauthorized:
		return fmt.Errorf("✗ Webhook rejected: invalid signature\n\nPlease check your UNKEY_GITHUB_APP_WEBHOOK_SECRET matches the ctrl-api configuration. By default its 'supersecret'")

	case http.StatusBadRequest:
		return fmt.Errorf("✗ Webhook rejected: invalid payload\n\n%s", string(bodyBytes))

	case http.StatusInternalServerError:
		return fmt.Errorf("✗ Server error: %s\n\n%s", resp.Status, string(bodyBytes))

	default:
		return fmt.Errorf("✗ Unexpected response: %s\n\n%s", resp.Status, string(bodyBytes))
	}
}

func ensureGithubConnection(ctx context.Context, databaseURL, projectID string, installationID, repositoryID int64, repository string) error {
	database, err := db.New(db.Config{
		PrimaryDSN:  databaseURL,
		ReadOnlyDSN: "",
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		_ = database.Close()
	}()

	// Try to insert, ignore if already exists
	err = db.Query.InsertGithubRepoConnection(ctx, database.RW(), db.InsertGithubRepoConnectionParams{
		ProjectID:          projectID,
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

func fetchRepositoryID(ctx context.Context, repository string) (int64, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s", repository)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "unkey-cli")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch repository: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return 0, fmt.Errorf("repository not found: %s\n\nThis tool only works with public repositories. If '%s' is private, make it public or use the actual repository ID directly", repository, repository)
		}
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return 0, fmt.Errorf("authentication required to access repository: %s\n\nThis tool only works with public repositories", repository)
		}
		return 0, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.ID, nil
}

func checkAllowUnauthenticatedDeployments() bool {
	// Check dev/.env.github file as fallback
	envPath := filepath.Join("dev", ".env.github")
	data, err := os.ReadFile(envPath)
	if err != nil {
		return false // Conservative default
	}

	// Simple parsing for UNKEY_ALLOW_UNAUTHENTICATED_DEPLOYMENTS=true
	lines := strings.SplitSeq(string(data), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		if val, ok := strings.CutPrefix(line, "UNKEY_ALLOW_UNAUTHENTICATED_DEPLOYMENTS="); ok {
			return strings.ToLower(strings.TrimSpace(val)) == "true"
		}
	}

	return false
}
