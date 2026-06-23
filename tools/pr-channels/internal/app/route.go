package app

import (
	"fmt"
	"strings"
)

// maxChannelName is Slack's hard limit on channel name length.
const maxChannelName = 80

// minBranchChars is the shortest branch fragment worth appending; below this
// the name is just `pr-<repo>-<number>`.
const minBranchChars = 3

// ChannelName derives the deterministic Slack channel name for a PR:
// `pr-<repo>-<number>-<branch>`. It is a pure function of its inputs (a PR's
// head branch is fixed for its lifetime), so handlers never need to look up or
// store a name to find a PR's channel again. The branch is slugified and is
// readability only — the `pr-<repo>-<number>` part is always preserved, and the
// branch is truncated or dropped to fit Slack's 80-character limit.
func ChannelName(repo string, number int, branch string) string {
	prefix := "pr-" + sanitize(repo)
	suffix := fmt.Sprintf("-%d", number)

	// The PR number identifies the channel; never drop it. If even
	// pr-<repo>-<number> overflows, trim the repo and skip the branch.
	if len(prefix)+len(suffix) > maxChannelName {
		keep := maxChannelName - len("pr-") - len(suffix)
		return "pr-" + sanitize(repo)[:keep] + suffix
	}

	base := prefix + suffix
	b := sanitize(branch)
	if b == "" {
		return base
	}
	budget := maxChannelName - len(base) - 1 // -1 for the joining hyphen
	if budget < minBranchChars {
		return base
	}
	if len(b) > budget {
		b = strings.Trim(b[:budget], "-_")
	}
	if b == "" {
		return base
	}
	return base + "-" + b
}

// sanitize maps an arbitrary string onto Slack's channel-name charset:
// lowercase letters, digits, hyphens and underscores only.
func sanitize(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// IsBotAuthor reports whether a PR author is an automation account whose PRs
// should not get a review channel (dependabot, renovate, GitHub Apps, ...).
func IsBotAuthor(login, authorType string) bool {
	if strings.EqualFold(authorType, "Bot") {
		return true
	}
	l := strings.ToLower(login)
	if strings.HasSuffix(l, "[bot]") {
		return true
	}
	switch l {
	case "dependabot", "dependabot-preview", "renovate", "renovate-bot":
		return true
	}
	return false
}

// ShouldOpenChannel reports whether a pull_request event should create a per-PR
// review channel. A channel is created only when there is someone to review:
//   - review_requested: always (a reviewer was just asked)
//   - opened/reopened/ready_for_review: only if reviewers are already attached
//
// Drafts (except via review_requested) and bot PRs never get a channel.
func ShouldOpenChannel(action string, draft, isBot, hasReviewers bool) bool {
	if isBot {
		return false
	}
	switch action {
	case "review_requested":
		return true
	case "opened", "reopened":
		return !draft && hasReviewers
	case "ready_for_review":
		return hasReviewers
	default:
		return false
	}
}

// ShouldPostOverview reports whether a pull_request event should post to the
// overview feed. The feed tracks PRs becoming ready for review (non-draft
// opened, or draft marked ready). Merges are handled separately; closes without
// merge are intentionally ignored. Bot PRs are excluded.
func ShouldPostOverview(action string, draft, isBot bool) bool {
	if isBot {
		return false
	}
	switch action {
	case "opened", "reopened":
		return !draft
	case "ready_for_review":
		return true
	default:
		return false
	}
}

// Mention renders a GitHub login as a Slack mention when a mapping exists, and
// falls back to plain text (still readable, just no ping) when it does not.
func Mention(login, slackUserID string) string {
	if slackUserID != "" {
		return "<@" + slackUserID + ">"
	}
	return "`@" + login + "`"
}
