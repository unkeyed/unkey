package keyleak

import (
	"context"

	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/svc/ctrl/internal/slack"
)

type memberResolver interface {
	MemberEmails(context.Context, string) ([]string, error)
}

type Service struct {
	hydrav1.UnimplementedKeyLeakServiceServer
	members         memberResolver
	email           email.Sender
	slack           *slack.Client
	slackWebhookURL string
}

type Config struct {
	Members         memberResolver
	Email           email.Sender
	SlackWebhookURL string
}

func New(cfg Config) *Service {
	return &Service{
		UnimplementedKeyLeakServiceServer: hydrav1.UnimplementedKeyLeakServiceServer{},
		members:                           cfg.Members,
		email:                             cfg.Email,
		slack:                             slack.NewClient(),
		slackWebhookURL:                   cfg.SlackWebhookURL,
	}
}
