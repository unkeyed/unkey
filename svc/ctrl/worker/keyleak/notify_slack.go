package keyleak

import (
	"context"
	"fmt"
	"net/mail"
	"net/url"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/svc/ctrl/internal/slack"
)

func (s *Service) NotifySlack(ctx restate.Context, req *hydrav1.KeyLeakNotification) (*hydrav1.KeyLeakResponse, error) {
	logger := ctx.Log()
	err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		if s.slackWebhookURL == "" {
			return nil
		}
		lookupCtx, cancel := context.WithTimeout(rc, 3*time.Second)
		defer cancel()
		members, err := s.members.MemberEmails(lookupCtx, req.GetOrgId())
		memberDetails := "Workspace members unavailable"
		if err != nil {
			logger.Warn("could not resolve workspace members for key leak Slack alert", "org_id", req.GetOrgId(), "error", err)
		} else {
			memberDetails = slackMembers(members)
		}
		return s.slack.Send(rc, s.slackWebhookURL, slackMessage(req, memberDetails))
	}, runOptions("send key leak Slack alert")...)

	return &hydrav1.KeyLeakResponse{}, err
}

func slackMessage(req *hydrav1.KeyLeakNotification, members string) slack.Payload {
	title := "Key exposed on GitHub"
	switch req.GetType() {
	case "unkey_root_key":
		title = "Root key exposed on GitHub"
	case "unkey_key":
		title = "API key exposed on GitHub"
	}
	workspace := slackText(req.GetWorkspaceName())
	keyID := slackText(req.GetKeyId())
	source := slackSource(req.GetUrl(), req.GetSource())
	return slack.Payload{
		Text: fmt.Sprintf("%s. Workspace: %s. Key ID: %s. Rotate or revoke this key.\n%s", title, workspace, keyID, source),
		Blocks: []slack.Block{
			slack.NewHeaderBlock(title),
			slack.NewMarkdownBlock("Rotate or revoke this key."),
			slack.NewSectionBlock(
				slack.Field{Type: "mrkdwn", Text: "*Workspace*\n" + workspace, Verbatim: true},
				slack.Field{Type: "mrkdwn", Text: "*Key ID*\n`" + keyID + "`", Verbatim: true},
			),
			slack.NewMarkdownBlock(members),
			slack.NewMarkdownBlock("*Source*\n" + source),
			slack.NewContextBlock(fmt.Sprintf("Reported %s · GitHub %s", slackText(req.GetDate()), slackText(req.GetSource()))),
			slack.NewContextBlock("Organization ID: " + slackText(req.GetOrgId())),
		},
	}
}

func slackMembers(emails []string) string {
	if len(emails) == 0 {
		return "No member email addresses found"
	}
	text := "*Workspace members*\n"
	shown := 0
	for _, email := range emails[:min(3, len(emails))] {
		member := slackText(email)
		if address, err := mail.ParseAddress(email); err == nil && address.Address == email && len(email) <= 254 {
			link := strings.ReplaceAll(url.PathEscape(email), "&", "&amp;")
			member = fmt.Sprintf("<mailto:%s|%s>", link, member)
		}
		if len(text)+len(member) > 2800 {
			break
		}
		if shown > 0 {
			text += ", "
		}
		text += member
		shown++
	}
	if remaining := len(emails) - shown; remaining > 0 {
		text += fmt.Sprintf("\nand %d more members", remaining)
	}
	return text
}

func slackSource(rawURL, source string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" || parsed.User != nil {
		return "Source URL unavailable"
	}
	link := strings.NewReplacer("&", "&amp;", "<", "%3C", ">", "%3E", "|", "%7C", " ", "%20").Replace(parsed.String())
	if len(link) > 2000 {
		return "Source URL unavailable"
	}
	label := "View source"
	if source == "content" {
		label = "View exposed file"
	}
	text := fmt.Sprintf("<%s|%s>", link, label)
	parts := strings.SplitN(strings.TrimPrefix(parsed.Path, "/"), "/", 5)
	if parsed.Host == "github.com" && len(parts) == 5 && parts[2] == "blob" {
		summary := slackText(parts[0] + "/" + parts[1] + " / " + parts[4])
		if len(text)+len(summary) < 2900 {
			text += "\n" + summary
		}
	}
	return text
}

func slackText(text string) string {
	runes := []rune(text)
	if len(runes) > 250 {
		text = string(runes[:249]) + "…"
	}
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(text)
}
