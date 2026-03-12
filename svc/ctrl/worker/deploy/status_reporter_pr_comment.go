package deploy

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/logger"
	githubclient "github.com/unkeyed/unkey/svc/ctrl/worker/github"
)

const (
	// prCommentMainMarker identifies the shared deployment comment on a PR.
	prCommentMainMarker = "<!-- unkey-deploy -->"

	// prCommentRowMarkerFmt wraps each app/env's table row for find-and-replace.
	// Keyed by app+env so a new deploy replaces the previous row for the same app.
	prCommentRowMarkerFmt = "<!-- row:%s:%s -->"
)

// rowPattern matches a full table row line that starts with a row marker.
var rowPattern = regexp.MustCompile(`(?m)^\| <!-- row:\S+ --> .+\|$`)

// prCommentReporter creates and updates a shared PR comment with one row per
// deployment, similar to the Vercel deployment comment. Multiple deploy
// workflows running concurrently for the same PR each manage their own row.
// All GitHub API calls are fire-and-forget.
type prCommentReporter struct {
	github         githubclient.GitHubClient
	installationID int64
	repo           string
	branch         string
	commitSHA      string
	deploymentID   string
	projectSlug    string
	appSlug        string
	envSlug        string
	logURL         string
	environmentURL string

	prNumber  int   // resolved lazily
	commentID int64 // set after Create
}

type prCommentReporterConfig struct {
	GitHub         githubclient.GitHubClient
	InstallationID int64
	Repo           string
	Branch         string
	CommitSHA      string
	DeploymentID   string
	ProjectSlug    string
	AppSlug        string
	EnvSlug        string
	LogURL         string
	EnvironmentURL string
}

func newPRCommentReporter(cfg prCommentReporterConfig) *prCommentReporter {
	return &prCommentReporter{
		github:         cfg.GitHub,
		installationID: cfg.InstallationID,
		repo:           cfg.Repo,
		branch:         cfg.Branch,
		commitSHA:      cfg.CommitSHA,
		deploymentID:   cfg.DeploymentID,
		projectSlug:    cfg.ProjectSlug,
		appSlug:        cfg.AppSlug,
		envSlug:        cfg.EnvSlug,
		logURL:         cfg.LogURL,
		environmentURL: cfg.EnvironmentURL,
	}
}

// Create looks up the PR, finds or creates the shared comment, and adds this
// deployment's row.
func (r *prCommentReporter) Create(ctx restate.ObjectSharedContext) {
	if r.installationID == 0 || r.repo == "" || r.branch == "" {
		return
	}

	prNumber, err := restate.Run(ctx, func(_ restate.RunContext) (int, error) {
		return r.github.FindPullRequestForBranch(r.installationID, r.repo, r.branch)
	}, restate.WithName("find PR for branch"), restate.WithMaxRetryDuration(30*time.Second))
	if err != nil || prNumber == 0 {
		if err != nil {
			logger.Error("failed to find PR for branch", "error", err, "branch", r.branch)
		}
		return
	}
	r.prNumber = prNumber

	// Look for an existing deployment comment on this PR.
	existing, err := restate.Run(ctx, func(_ restate.RunContext) (findResult, error) {
		id, body, err := r.github.FindBotComment(r.installationID, r.repo, r.prNumber, prCommentMainMarker)
		return findResult{ID: id, Body: body}, err
	}, restate.WithName("find existing deploy comment"), restate.WithMaxRetryDuration(30*time.Second))
	if err != nil {
		logger.Error("failed to search for existing deploy comment", "error", err)
	}

	row := r.buildRow("⏳ Queued")

	if existing.ID != 0 {
		// Add or replace our row in the existing comment.
		r.commentID = existing.ID
		body := r.upsertRow(existing.Body, row)

		_ = restate.RunVoid(ctx, func(_ restate.RunContext) error {
			return r.github.UpdateIssueComment(r.installationID, r.repo, r.commentID, body)
		}, restate.WithName("add row to existing deploy comment"), restate.WithMaxRetryDuration(30*time.Second))
		return
	}

	// No existing comment — create one.
	body := r.buildFullComment(row)
	commentID, err := restate.Run(ctx, func(_ restate.RunContext) (int64, error) {
		return r.github.CreateIssueComment(r.installationID, r.repo, r.prNumber, body)
	}, restate.WithName("create PR deployment comment"), restate.WithMaxRetryDuration(30*time.Second))
	if err != nil {
		logger.Error("failed to create PR comment", "error", err, "pr", r.prNumber)
		return
	}
	r.commentID = commentID
}

// findResult is a helper to return both ID and Body from restate.Run.
type findResult struct {
	ID   int64
	Body string
}

// Report updates this deployment's row in the shared PR comment.
func (r *prCommentReporter) Report(ctx restate.ObjectSharedContext, state string, description string) {
	if r.commentID == 0 {
		return
	}

	statusEmoji, statusLabel := r.stateToDisplay(state)
	row := r.buildRow(statusEmoji + " " + statusLabel)

	// Re-read current comment body so we don't clobber other deployments' rows.
	current, err := restate.Run(ctx, func(_ restate.RunContext) (findResult, error) {
		id, body, err := r.github.FindBotComment(r.installationID, r.repo, r.prNumber, prCommentMainMarker)
		return findResult{ID: id, Body: body}, err
	}, restate.WithName("read deploy comment for update"), restate.WithMaxRetryDuration(30*time.Second))
	if err != nil || current.ID == 0 {
		return
	}

	body := r.upsertRow(current.Body, row)

	_ = restate.RunVoid(ctx, func(_ restate.RunContext) error {
		return r.github.UpdateIssueComment(r.installationID, r.repo, r.commentID, body)
	}, restate.WithName(fmt.Sprintf("update PR comment row: %s", state)), restate.WithMaxRetryDuration(30*time.Second))
}

// rowMarker returns the unique key for this app/env combination.
func (r *prCommentReporter) rowMarker() string {
	return fmt.Sprintf(prCommentRowMarkerFmt, r.appSlug, r.envSlug)
}

// buildRow produces a single markdown table row with this deployment's info.
func (r *prCommentReporter) buildRow(status string) string {
	rowMarker := r.rowMarker()

	nameLabel := r.projectSlug
	if r.appSlug != "default" {
		nameLabel += " / " + r.appSlug
	}

	now := time.Now().UTC().Format("Jan 2, 2006 3:04pm")

	preview := "—"
	if r.environmentURL != "" {
		preview = fmt.Sprintf("[Visit Preview](%s)", r.environmentURL)
	}

	inspect := fmt.Sprintf("[Inspect](%s)", r.logURL)

	return fmt.Sprintf("| %s **%s** (%s) | %s | %s | %s | %s |",
		rowMarker, nameLabel, r.envSlug, status, preview, inspect, now)
}

// buildFullComment wraps the header, table, and a single row into a new comment.
func (r *prCommentReporter) buildFullComment(firstRow string) string {
	var b strings.Builder
	b.WriteString(prCommentMainMarker)
	b.WriteString("\n")
	b.WriteString("**The latest updates on your projects.** Learn more about [Unkey Deploy](https://www.unkey.com/docs/deployments)\n\n")
	b.WriteString("| Name | Status | Preview | Inspect | Updated (UTC) |\n")
	b.WriteString("|:--|:--|:--|:--|:--|\n")
	b.WriteString(firstRow)
	b.WriteString("\n")
	return b.String()
}

// upsertRow replaces an existing row for this app/env or appends a new one.
func (r *prCommentReporter) upsertRow(existingBody string, newRow string) string {
	rowMarker := r.rowMarker()

	if strings.Contains(existingBody, rowMarker) {
		// Replace the existing row (the whole line containing our marker).
		lines := strings.Split(existingBody, "\n")
		for i, line := range lines {
			if strings.Contains(line, rowMarker) {
				lines[i] = newRow
				break
			}
		}
		return strings.Join(lines, "\n")
	}

	// Append new row after the last table row.
	// Find the last line that starts with "| " (table row).
	lines := strings.Split(existingBody, "\n")
	lastRowIdx := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "|") && !strings.Contains(line, ":--") && i > 0 {
			lastRowIdx = i
		}
	}

	if lastRowIdx >= 0 {
		// Insert after the last row.
		result := make([]string, 0, len(lines)+1)
		result = append(result, lines[:lastRowIdx+1]...)
		result = append(result, newRow)
		result = append(result, lines[lastRowIdx+1:]...)
		return strings.Join(result, "\n")
	}

	// Fallback: just append.
	return existingBody + newRow + "\n"
}

func (r *prCommentReporter) stateToDisplay(state string) (string, string) {
	switch state {
	case "pending":
		return "⏳", "Queued"
	case "in_progress":
		return "🔨", "Building"
	case "success":
		return "✅", "Ready"
	case "failure", "error":
		return "❌", "Failed"
	default:
		return "⏳", "In Progress"
	}
}
