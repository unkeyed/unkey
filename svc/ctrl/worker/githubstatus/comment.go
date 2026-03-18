package githubstatus

import (
	"fmt"
	"strings"
	"time"
)

const (
	// prCommentMainMarker identifies the shared deployment comment on a PR.
	prCommentMainMarker = "<!-- unkey-deploy -->"

	// prCommentRowMarkerFmt wraps each app/env's table row for find-and-replace.
	prCommentRowMarkerFmt = "<!-- row:%s:%s -->"
)

func rowMarker(appSlug, envSlug string) string {
	return fmt.Sprintf(prCommentRowMarkerFmt, appSlug, envSlug)
}

func buildRow(projectSlug, appSlug, envSlug, environmentURL, logURL, status string) string {
	nameLabel := projectSlug
	if appSlug != "default" {
		nameLabel += " / " + appSlug
	}

	preview := "—"
	if environmentURL != "" {
		preview = fmt.Sprintf("[Visit Preview](%s)", environmentURL)
	}

	return fmt.Sprintf("| %s **%s** (%s) | %s | %s | [Inspect](%s) | %s |",
		rowMarker(appSlug, envSlug), nameLabel, envSlug, status,
		preview, logURL,
		time.Now().UTC().Format("Jan 2, 2006 3:04pm"))
}

func buildFullComment(firstRow string) string {
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
func upsertRow(appSlug, envSlug, body, newRow string) string {
	marker := rowMarker(appSlug, envSlug)
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
