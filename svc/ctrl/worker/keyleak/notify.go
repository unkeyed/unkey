package keyleak

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/svc/ctrl/internal/slack"
)

func (s *Service) Notify(ctx restate.Context, req *hydrav1.KeyLeakNotification) (*hydrav1.KeyLeakResponse, error) {
	client := hydrav1.NewKeyLeakServiceClient(ctx)
	client.NotifySlack().Send(req, restate.WithIdempotencyKey("leaked-key:"+req.GetKeyId()))

	recipients, err := restate.Run(ctx, func(rc restate.RunContext) ([]string, error) {
		return s.members.MemberEmails(rc, req.GetOrgId())
	}, restate.WithName("resolve key leak recipients"))
	if err != nil {
		return nil, err
	}
	for _, recipient := range recipients {
		client.NotifyEmail().Send(&hydrav1.KeyLeakEmailRequest{Leak: req, Email: recipient},
			restate.WithIdempotencyKey(emailIdempotencyKey(req.GetKeyId(), recipient)))
	}
	return &hydrav1.KeyLeakResponse{}, nil
}

func (s *Service) NotifyEmail(ctx restate.Context, req *hydrav1.KeyLeakEmailRequest) (*hydrav1.KeyLeakResponse, error) {
	err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return s.email.Send(rc, email.Email{
			From:       "",
			To:         []string{req.GetEmail()},
			Subject:    "",
			TemplateID: "secret-scanning-key-detected",
			Variables: map[string]string{
				"DATE":   req.GetLeak().GetDate(),
				"SOURCE": req.GetLeak().GetSource(),
				"URL":    req.GetLeak().GetUrl(),
			},
			IdempotencyKey: emailIdempotencyKey(req.GetLeak().GetKeyId(), req.GetEmail()),
		})
	}, restate.WithName("send key leak email"))
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
	}, restate.WithName("send key leak Slack alert"))
	return &hydrav1.KeyLeakResponse{}, err
}

func emailIdempotencyKey(keyID, recipient string) string {
	return "leaked-key:" + keyID + ":" + hash.Sha256(recipient)
}
