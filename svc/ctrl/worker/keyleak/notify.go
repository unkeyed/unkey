package keyleak

import (
	"context"
	"errors"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/svc/ctrl/internal/slack"
)

func (s *Service) Notify(ctx restate.Context, req *hydrav1.KeyLeakNotification) (*hydrav1.KeyLeakResponse, error) {
	client := hydrav1.NewKeyLeakServiceClient(ctx)
	if s.slackWebhookURL != "" {
		client.NotifySlack().Send(req)
	}
	if req.GetOrgId() == "" {
		return nil, restate.ToTerminalError(errors.New("leaked key has no organization"))
	}

	recipients, err := restate.Run(ctx, func(rc restate.RunContext) ([]string, error) {
		lookupCtx, cancel := context.WithTimeout(rc, 10*time.Second)
		defer cancel()

		return s.members.MemberEmails(lookupCtx, req.GetOrgId())
	}, runOptions("resolve key leak recipients")...)
	if err != nil {
		return nil, err
	}

	for _, recipient := range recipients {
		client.NotifyEmail().Send(&hydrav1.KeyLeakEmailRequest{Leak: req, Email: recipient})
	}
	return &hydrav1.KeyLeakResponse{}, nil
}

func (s *Service) NotifyEmail(ctx restate.Context, req *hydrav1.KeyLeakEmailRequest) (*hydrav1.KeyLeakResponse, error) {
	idempotencyKey := "leaked-key:" + ctx.Request().ID
	err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		sendCtx, cancel := context.WithTimeout(rc, 10*time.Second)
		defer cancel()

		return s.email.Send(sendCtx, email.Email{
			From:       "",
			To:         []string{req.GetEmail()},
			Subject:    "",
			TemplateID: "secret-scanning-key-detected",
			Variables: map[string]string{
				"DATE":   req.GetLeak().GetDate(),
				"SOURCE": req.GetLeak().GetSource(),
				"URL":    req.GetLeak().GetUrl(),
			},
			IdempotencyKey: idempotencyKey,
		})
	}, runOptions("send key leak email")...)
	return &hydrav1.KeyLeakResponse{}, err
}

func (s *Service) NotifySlack(ctx restate.Context, req *hydrav1.KeyLeakNotification) (*hydrav1.KeyLeakResponse, error) {
	err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return s.slack.Send(rc, s.slackWebhookURL, slack.Payload{
			Text: "Leaked Key Found",
			Blocks: []slack.Block{slack.NewSectionBlock(
				slack.Field{Type: "plain_text", Text: fmt.Sprintf("Type: %s\nSource: %s\nDate: %s\nURL: %s", req.GetType(), req.GetSource(), req.GetDate(), req.GetUrl())},
				slack.Field{Type: "plain_text", Text: fmt.Sprintf("Key: %s\nWorkspace: %s\nTenant: %s", req.GetKeyId(), req.GetWorkspaceName(), req.GetOrgId())},
			)},
		})
	}, runOptions("send key leak Slack alert")...)
	return &hydrav1.KeyLeakResponse{}, err
}
