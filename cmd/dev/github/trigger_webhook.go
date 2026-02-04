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
	"time"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

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
		cli.String("webhook-secret", "Secret for signing", cli.Default("local-dev-secret"), cli.EnvVar("UNKEY_GITHUB_APP_WEBHOOK_SECRET")),
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

	// Build payload
	payload := buildPushPayload(pushPayloadInput{
		Branch:         branch,
		CommitSHA:      commitSHA,
		CommitMessage:  "Manual deployment trigger",
		InstallationID: installationID,
		RepositoryID:   repositoryID,
		Repository:     repository,
		AuthorName:     "Developer",
		AuthorUsername: "dev",
	})

	// Serialize to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to serialize payload: %w", err)
	}

	// Generate signature
	signature := generateSignature(payloadBytes, webhookSecret)

	// Send request
	fmt.Println("\nTriggering deployment webhook...")

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(payloadBytes))
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

	bodyBytes, _ := io.ReadAll(resp.Body)

	// Handle response
	switch resp.StatusCode {
	case http.StatusOK:
		fmt.Println("✔ Webhook delivered successfully")
		fmt.Println("✔ Deployment created")
		fmt.Println()
		fmt.Printf("Repository: %s\n", repository)
		fmt.Printf("Branch: %s\n", branch)
		fmt.Printf("Commit: %s (%s)\n", commitSHA[:min(8, len(commitSHA))], "Manual deployment trigger")

		// Determine environment
		env := "preview"
		if branch == "main" || branch == "master" {
			env = "production"
		}
		fmt.Printf("Environment: %s\n", env)
		return nil

	case http.StatusUnauthorized:
		fmt.Println("✗ Webhook rejected: invalid signature")
		fmt.Println()
		fmt.Println("Please check your UNKEY_GITHUB_APP_WEBHOOK_SECRET matches the ctrl-api configuration.")
		os.Exit(1)

	case http.StatusBadRequest:
		fmt.Printf("✗ Webhook rejected: invalid payload\n\n%s\n", string(bodyBytes))
		os.Exit(1)

	case http.StatusInternalServerError:
		fmt.Printf("✗ Server error: %s\n\n%s\n", resp.Status, string(bodyBytes))
		os.Exit(1)

	default:
		fmt.Printf("✗ Unexpected response: %s\n\n%s\n", resp.Status, string(bodyBytes))
		os.Exit(1)
	}

	return nil
}

func ensureGithubConnection(ctx context.Context, databaseURL, projectID string, installationID, repositoryID int64, repository string) error {
	logger := logging.New()

	database, err := db.New(db.Config{
		PrimaryDSN:  databaseURL,
		ReadOnlyDSN: "",
		Logger:      logger,
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
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
