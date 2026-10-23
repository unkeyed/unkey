package keyleak

import (
	"context"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/svc/ctrl/internal/slack"
)

type memberResolver interface {
	MemberEmails(context.Context, string) ([]string, error)
}

type slackSender interface {
	Send(context.Context, string, slack.Payload) error
}

type Service struct {
	hydrav1.UnimplementedKeyLeakServiceServer
	members         memberResolver
	email           email.Sender
	slack           slackSender
	slackWebhookURL string
}

type Config struct {
	Members         memberResolver
	Email           email.Sender
	SlackWebhookURL string
}

func New(cfg Config) *Service {
	var slackClient slackSender = slack.NewNoop()
	if cfg.SlackWebhookURL != "" {
		slackClient = slack.NewClient()
	}

	return &Service{
		UnimplementedKeyLeakServiceServer: hydrav1.UnimplementedKeyLeakServiceServer{},
		members:                           cfg.Members,
		email:                             cfg.Email,
		slack:                             slackClient,
		slackWebhookURL:                   cfg.SlackWebhookURL,
	}
}

func runOptions(name string) []restate.RunOption {
	return []restate.RunOption{
		restate.WithName(name),
		restate.WithInitialRetryInterval(time.Second),
		restate.WithMaxRetryInterval(time.Minute),
		restate.WithMaxRetryAttempts(360),
		restate.WithMaxRetryDuration(6 * time.Hour),
	}
}
