package deploy

import (
	"fmt"
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

	// ghRetryDuration is the max retry duration for GitHub API calls via restate.
	// PR comments are best-effort — fail fast rather than block the deploy.
	ghRetryDuration = 5 * time.Second
)

// findResult bundles the comment ID and body for restate.Run serialisation.
type findResult struct {
	ID   int64
	Body string
}

// prCommentReporter creates and updates a shared PR comment with one row per
// app/env combination. Multiple deploy workflows for the same PR each manage
// their own row. All GitHub API calls are fire-and-forget.
type prCommentReporter struct {
	prCommentReporterConfig

	prNumber  int
	commentID int64
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
	return &prCommentReporter{prCommentReporterConfig: cfg}
}

func (r *prCommentReporter) findComment(ctx restate.ObjectSharedContext, name string) (findResult, error) {
	return restate.Run(ctx, func(_ restate.RunContext) (findResult, error) {
		id, body, err := r.GitHub.FindBotComment(r.InstallationID, r.Repo, r.prNumber, prCommentMainMarker)
		return findResult{ID: id, Body: body}, err
	}, restate.WithName(name), restate.WithMaxRetryDuration(ghRetryDuration))
}

func (r *prCommentReporter) updateComment(ctx restate.ObjectSharedContext, body string, name string) {
	_ = restate.RunVoid(ctx, func(_ restate.RunContext) error {
		return r.GitHub.UpdateIssueComment(r.InstallationID, r.Repo, r.commentID, body)
	}, restate.WithName(name), restate.WithMaxRetryDuration(ghRetryDuration))
}

func (r *prCommentReporter) Create(ctx restate.ObjectSharedContext) {
	if r.InstallationID == 0 || r.Repo == "" || r.Branch == "" {
		return
	}

	prNumber, err := restate.Run(ctx, func(_ restate.RunContext) (int, error) {
		return r.GitHub.FindPullRequestForBranch(r.InstallationID, r.Repo, r.Branch)
	}, restate.WithName("find PR for branch"), restate.WithMaxRetryDuration(ghRetryDuration))
	if err != nil || prNumber == 0 {
		if err != nil {
			logger.Error("failed to find PR for branch", "error", err, "branch", r.Branch)
		}
		return
	}
	r.prNumber = prNumber

	existing, err := r.findComment(ctx, "find existing deploy comment")
	if err != nil {
		logger.Error("failed to search for existing deploy comment", "error", err)
	}

	row := r.buildRow("Queued")

	if existing.ID != 0 {
		r.commentID = existing.ID
		r.updateComment(ctx, r.upsertRow(existing.Body, row), "add row to deploy comment")
		return
	}

	body := r.buildFullComment(row)
	commentID, createErr := restate.Run(ctx, func(_ restate.RunContext) (int64, error) {
		return r.GitHub.CreateIssueComment(r.InstallationID, r.Repo, r.prNumber, body)
	}, restate.WithName("create deploy comment"), restate.WithMaxRetryDuration(ghRetryDuration))
	if createErr != nil {
		logger.Error("failed to create PR comment", "error", createErr, "pr", r.prNumber)
		return
	}
	r.commentID = commentID
}

func (r *prCommentReporter) Report(ctx restate.ObjectSharedContext, state string, description string) {
	if r.commentID == 0 {
		return
	}

	row := r.buildRow(stateLabel(state))

	// Re-read current body so we don't clobber other apps' rows.
	current, err := r.findComment(ctx, "read deploy comment")
	if err != nil || current.ID == 0 {
		return
	}

	r.updateComment(ctx, r.upsertRow(current.Body, row), fmt.Sprintf("update deploy comment: %s", state))
}

func (r *prCommentReporter) rowMarker() string {
	return fmt.Sprintf(prCommentRowMarkerFmt, r.AppSlug, r.EnvSlug)
}

func (r *prCommentReporter) buildRow(status string) string {
	nameLabel := r.ProjectSlug
	if r.AppSlug != "default" {
		nameLabel += " / " + r.AppSlug
	}

	preview := "—"
	if r.EnvironmentURL != "" {
		preview = fmt.Sprintf("[Visit Preview](%s)", r.EnvironmentURL)
	}

	return fmt.Sprintf("| %s **%s** (%s) | %s | %s | [Inspect](%s) | %s |",
		r.rowMarker(), nameLabel, r.EnvSlug, status,
		preview, r.LogURL,
		time.Now().UTC().Format("Jan 2, 2006 3:04pm"))
}

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
func (r *prCommentReporter) upsertRow(body string, newRow string) string {
	marker := r.rowMarker()
	lines := strings.Split(body, "\n")

	for i, line := range lines {
		if strings.Contains(line, marker) {
			lines[i] = newRow
			return strings.Join(lines, "\n")
		}
	}

	// Append after the last table row (any line starting with "|" that isn't the separator).
	lastRowIdx := -1
	for i, line := range lines {
		if i > 0 && strings.HasPrefix(line, "|") && !strings.Contains(line, ":--") {
			lastRowIdx = i
		}
	}

	if lastRowIdx >= 0 {
		result := make([]string, 0, len(lines)+1)
		result = append(result, lines[:lastRowIdx+1]...)
		result = append(result, newRow)
		result = append(result, lines[lastRowIdx+1:]...)
		return strings.Join(result, "\n")
	}

	return body + newRow + "\n"
}

func stateLabel(state string) string {
	switch state {
	case "pending":
		return "Queued"
	case "in_progress":
		return "Building"
	case "success":
		return "Ready"
	case "failure", "error":
		return "Failed"
	default:
		return "In Progress"
	}
}
