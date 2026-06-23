// Package app holds the event->action state machine that turns GitHub webhook
// events into Slack channel lifecycle operations.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/go-github/v66/github"

	"github.com/unkeyed/unkey/tools/pr-channels/internal/config"
	"github.com/unkeyed/unkey/tools/pr-channels/internal/slackclient"
	"github.com/unkeyed/unkey/tools/pr-channels/internal/store"
)

// App wires together config, persistence and the Slack client.
type App struct {
	cfg   config.Config
	store *store.Store
	slack *slackclient.Client
	log   *slog.Logger
}

// New constructs an App.
func New(cfg config.Config, st *store.Store, sc *slackclient.Client, log *slog.Logger) *App {
	return &App{cfg: cfg, store: st, slack: sc, log: log}
}

// Handle dispatches a parsed GitHub webhook event.
func (a *App) Handle(ctx context.Context, event any) error {
	switch e := event.(type) {
	case *github.PullRequestEvent:
		return a.onPullRequest(ctx, e)
	case *github.PullRequestReviewEvent:
		return a.onReview(ctx, e)
	case *github.IssueCommentEvent:
		return a.onIssueComment(ctx, e)
	case *github.CheckSuiteEvent:
		return a.onCheckSuite(ctx, e)
	default:
		return nil // event types we do not care about
	}
}

func (a *App) onPullRequest(ctx context.Context, e *github.PullRequestEvent) error {
	repo := e.GetRepo().GetName()
	if !a.cfg.ManagesRepo(repo) {
		return nil
	}
	pr := e.GetPullRequest()
	number := pr.GetNumber()
	action := e.GetAction()
	isBot := IsBotAuthor(pr.GetUser().GetLogin(), pr.GetUser().GetType())

	switch action {
	case "closed":
		return a.onClosed(ctx, repo, pr, isBot)

	case "review_requested":
		return a.onReviewRequested(ctx, repo, pr, e.GetRequestedReviewer(), isBot)

	case "synchronize":
		// New commits pushed: keep the PR fresh but do not spam the channel.
		return a.store.Touch(ctx, repo, number)

	case "opened", "reopened", "ready_for_review":
		return a.onReadyOrOpen(ctx, repo, pr, action, isBot)
	}
	return nil
}

// onReadyOrOpen posts to the overview feed and, when reviewers are already
// attached, opens the per-PR review channel. A PR with no requested reviewers
// gets a feed entry but no channel (it has nothing to review yet).
func (a *App) onReadyOrOpen(ctx context.Context, repo string, pr *github.PullRequest, action string, isBot bool) error {
	var channelID string
	if ShouldOpenChannel(action, pr.GetDraft(), isBot, len(reviewerLogins(pr)) > 0) {
		id, err := a.openChannel(ctx, repo, pr)
		if err != nil {
			return err
		}
		channelID = id
	}
	if ShouldPostOverview(action, pr.GetDraft(), isBot) {
		return a.postOverviewOpened(ctx, repo, pr, channelID)
	}
	return nil
}

// openChannel creates (or unarchives) the per-PR review channel, records the PR,
// invites the author and reviewers, and posts the opening summary. It returns
// the Slack channel ID so callers can reference it from the overview feed.
func (a *App) openChannel(ctx context.Context, repo string, pr *github.PullRequest) (string, error) {
	number := pr.GetNumber()
	name := ChannelName(repo, number, pr.GetHead().GetRef())

	channelID, err := a.slack.EnsureChannel(ctx, name)
	if err != nil {
		return "", fmt.Errorf("ensure channel %s: %w", name, err)
	}

	reviewers := reviewerLogins(pr)
	if err := a.store.UpsertPR(ctx, store.PR{
		Repo: repo, Number: number,
		ChannelID: channelID, ChannelName: name,
		State:     "open",
		Author:    pr.GetUser().GetLogin(),
		Reviewers: reviewers,
		URL:       pr.GetHTMLURL(),
		Title:     pr.GetTitle(),
	}); err != nil {
		return "", err
	}

	// Invite author + reviewers (only those we can resolve to Slack ids).
	logins := append([]string{pr.GetUser().GetLogin()}, reviewers...)
	ids, err := a.store.SlackIDs(ctx, logins)
	if err != nil {
		return "", err
	}
	if err := a.slack.Invite(ctx, channelID, values(ids)); err != nil {
		a.log.Warn("invite failed", "channel", name, "err", err)
	}

	summary := a.openSummary(pr, repo, reviewers, ids)
	if err := a.slack.Post(ctx, channelID, summary); err != nil {
		a.log.Warn("summary post failed", "channel", name, "err", err)
	}
	return channelID, nil
}

func (a *App) openSummary(pr *github.PullRequest, repo string, reviewers []string, ids map[string]string) string {
	var b strings.Builder
	fmt.Fprintf(&b, ":git: *<%s|#%d %s>*\n", pr.GetHTMLURL(), pr.GetNumber(), pr.GetTitle())
	fmt.Fprintf(&b, "Repo `%s` · opened by %s\n",
		repo, Mention(pr.GetUser().GetLogin(), ids[pr.GetUser().GetLogin()]))
	fmt.Fprintf(&b, "+%d −%d across %d files\n",
		pr.GetAdditions(), pr.GetDeletions(), pr.GetChangedFiles())
	if len(reviewers) > 0 {
		mentions := make([]string, 0, len(reviewers))
		for _, r := range reviewers {
			mentions = append(mentions, Mention(r, ids[r]))
		}
		fmt.Fprintf(&b, "Reviewers: %s", strings.Join(mentions, " "))
	} else {
		b.WriteString("_No reviewers requested yet._")
	}
	return b.String()
}

// onReviewRequested handles a reviewer being added. If the PR has no channel yet
// (e.g. it was opened with no reviewers, or as a draft), this is the moment we
// create one. If the channel already exists, we invite the new reviewer and
// record them so reminders ping the right people.
func (a *App) onReviewRequested(ctx context.Context, repo string, pr *github.PullRequest, requested *github.User, isBot bool) error {
	number := pr.GetNumber()
	tracked, err := a.store.GetPR(ctx, repo, number)
	if err != nil {
		return err
	}
	if tracked == nil {
		if !ShouldOpenChannel("review_requested", pr.GetDraft(), isBot, true) {
			return nil
		}
		_, err := a.openChannel(ctx, repo, pr)
		return err
	}

	login := requested.GetLogin()
	if login == "" {
		// Team review request or unparsable payload: nothing to ping.
		return a.store.Touch(ctx, repo, number)
	}
	if err := a.store.AddReviewer(ctx, repo, number, login); err != nil {
		return err
	}
	ids, _ := a.store.SlackIDs(ctx, []string{login})
	if err := a.slack.Invite(ctx, tracked.ChannelID, values(ids)); err != nil {
		a.log.Warn("invite failed", "channel", tracked.ChannelName, "err", err)
	}
	return a.slack.Post(ctx, tracked.ChannelID, "Review requested from "+Mention(login, ids[login]))
}

func (a *App) onReview(ctx context.Context, e *github.PullRequestReviewEvent) error {
	if e.GetAction() != "submitted" {
		return nil
	}
	repo := e.GetRepo().GetName()
	number := e.GetPullRequest().GetNumber()
	pr, err := a.store.GetPR(ctx, repo, number)
	if err != nil || pr == nil {
		return err
	}
	review := e.GetReview()
	reviewer := review.GetUser().GetLogin()
	ids, _ := a.store.SlackIDs(ctx, []string{reviewer})

	var verb string
	switch strings.ToLower(review.GetState()) {
	case "approved":
		verb = ":white_check_mark: approved"
	case "changes_requested":
		verb = ":arrows_counterclockwise: requested changes"
	case "commented":
		verb = ":speech_balloon: commented"
	default:
		verb = review.GetState()
	}
	msg := fmt.Sprintf("%s %s", Mention(reviewer, ids[reviewer]), verb)
	if body := strings.TrimSpace(review.GetBody()); body != "" {
		msg += "\n> " + strings.ReplaceAll(body, "\n", "\n> ")
	}
	_ = a.store.Touch(ctx, repo, number)
	return a.slack.Post(ctx, pr.ChannelID, msg)
}

func (a *App) onIssueComment(ctx context.Context, e *github.IssueCommentEvent) error {
	// Only top-level comments on PRs; skip issues and edits/deletes.
	if e.GetAction() != "created" || !e.GetIssue().IsPullRequest() {
		return nil
	}
	repo := e.GetRepo().GetName()
	number := e.GetIssue().GetNumber()
	pr, err := a.store.GetPR(ctx, repo, number)
	if err != nil || pr == nil {
		return err
	}
	author := e.GetComment().GetUser().GetLogin()
	ids, _ := a.store.SlackIDs(ctx, []string{author})
	body := strings.TrimSpace(e.GetComment().GetBody())
	msg := fmt.Sprintf("%s commented:\n> %s",
		Mention(author, ids[author]), strings.ReplaceAll(body, "\n", "\n> "))
	_ = a.store.Touch(ctx, repo, number)
	return a.slack.Post(ctx, pr.ChannelID, msg)
}

func (a *App) onCheckSuite(ctx context.Context, e *github.CheckSuiteEvent) error {
	cs := e.GetCheckSuite()
	if e.GetAction() != "completed" {
		return nil
	}
	// Only report failures; green CI is the expected case and would be noise.
	if cs.GetConclusion() == "success" || cs.GetConclusion() == "neutral" || cs.GetConclusion() == "skipped" {
		return nil
	}
	repo := e.GetRepo().GetName()
	for _, pr := range cs.PullRequests {
		tracked, err := a.store.GetPR(ctx, repo, pr.GetNumber())
		if err != nil || tracked == nil {
			continue
		}
		_ = a.slack.Post(ctx, tracked.ChannelID,
			fmt.Sprintf(":x: CI %s", cs.GetConclusion()))
	}
	return nil
}

// onClosed records a merge in the overview feed (closes without a merge are
// intentionally ignored there) and, when the PR has a review channel, posts the
// outcome and schedules the channel for archival.
func (a *App) onClosed(ctx context.Context, repo string, pr *github.PullRequest, isBot bool) error {
	number := pr.GetNumber()
	merged := pr.GetMerged()

	if merged && !isBot {
		if err := a.postOverviewMerged(ctx, repo, pr); err != nil {
			a.log.Warn("overview merged post failed", "repo", repo, "number", number, "err", err)
		}
	}

	tracked, err := a.store.GetPR(ctx, repo, number)
	if err != nil || tracked == nil {
		return err
	}
	outcome := ":wastebasket: closed without merging"
	if merged {
		outcome = ":tada: merged"
	}
	if err := a.slack.Post(ctx, tracked.ChannelID,
		outcome+" — archiving this channel shortly."); err != nil {
		a.log.Warn("close post failed", "channel", tracked.ChannelName, "err", err)
	}
	if err := a.store.SetClosed(ctx, repo, number, merged); err != nil {
		return err
	}
	// Archive after a short delay so people can still read the outcome. The
	// delay is best-effort within this process; a missed archive is harmless
	// and the channel can be archived by the next close event or by hand.
	go a.archiveLater(repo, number, tracked.ChannelID, tracked.ChannelName)
	return nil
}

func (a *App) archiveLater(repo string, number int, channelID, channelName string) {
	t := time.NewTimer(a.cfg.ArchiveDelay)
	defer t.Stop()
	<-t.C
	bg, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := a.slack.Archive(bg, channelID); err != nil {
		a.log.Warn("archive failed", "channel", channelName, "err", err)
		return
	}
	if err := a.store.MarkArchived(bg, repo, number); err != nil {
		a.log.Warn("mark archived failed", "channel", channelName, "err", err)
	}
}

// postOverview sends a line to the overview feed, when configured.
func (a *App) postOverview(ctx context.Context, markdown string) error {
	if a.cfg.OverviewChannelID == "" {
		return nil
	}
	return a.slack.Post(ctx, a.cfg.OverviewChannelID, markdown)
}

func (a *App) postOverviewOpened(ctx context.Context, repo string, pr *github.PullRequest, channelID string) error {
	msg := fmt.Sprintf(":git: *<%s|#%d %s>* opened by `@%s` in `%s`",
		pr.GetHTMLURL(), pr.GetNumber(), pr.GetTitle(), pr.GetUser().GetLogin(), repo)
	if channelID != "" {
		msg += fmt.Sprintf(" · review in <#%s>", channelID)
	}
	return a.postOverview(ctx, msg)
}

func (a *App) postOverviewMerged(ctx context.Context, repo string, pr *github.PullRequest) error {
	who := pr.GetMergedBy().GetLogin()
	if who == "" {
		who = pr.GetUser().GetLogin()
	}
	msg := fmt.Sprintf(":tada: *<%s|#%d %s>* merged by `@%s` in `%s`",
		pr.GetHTMLURL(), pr.GetNumber(), pr.GetTitle(), who, repo)
	return a.postOverview(ctx, msg)
}

// Remind nudges reviewers on PRs that have gone idle. Intended to be run on a
// schedule (Unkey Deploy cron or an external scheduler hitting `remind`).
func (a *App) Remind(ctx context.Context) error {
	stale, err := a.store.StaleOpenPRs(ctx, a.cfg.ReminderIdle)
	if err != nil {
		return err
	}
	for _, p := range stale {
		ids, _ := a.store.SlackIDs(ctx, p.Reviewers)
		mentions := make([]string, 0, len(p.Reviewers))
		for _, r := range p.Reviewers {
			mentions = append(mentions, Mention(r, ids[r]))
		}
		who := strings.Join(mentions, " ")
		if who == "" {
			who = "_no reviewers assigned_"
		}
		msg := fmt.Sprintf(":alarm_clock: This PR has been waiting %s. %s",
			a.cfg.ReminderIdle.Round(time.Hour), who)
		if err := a.slack.Post(ctx, p.ChannelID, msg); err != nil {
			a.log.Warn("reminder failed", "repo", p.Repo, "number", p.Number, "err", err)
			continue
		}
		_ = a.store.MarkReminded(ctx, p.Repo, p.Number)
	}
	a.log.Info("reminders sent", "count", len(stale))
	return nil
}

func reviewerLogins(pr *github.PullRequest) []string {
	out := make([]string, 0, len(pr.RequestedReviewers))
	for _, u := range pr.RequestedReviewers {
		if l := u.GetLogin(); l != "" {
			out = append(out, l)
		}
	}
	return out
}

func values(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for _, v := range m {
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}
