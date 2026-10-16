package keyleak

import (
	"context"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/slack"
)

type memberResolver interface {
	MemberEmails(context.Context, string) ([]string, error)
}

type Service struct {
	hydrav1.UnimplementedKeyLeakServiceServer
	db              db.Database
	members         memberResolver
	email           email.Sender
	slack           *slack.Client
	slackWebhookURL string
}

type Config struct {
	DB              db.Database
	Members         memberResolver
	Email           email.Sender
	SlackWebhookURL string
}

func New(cfg Config) *Service {
	return &Service{
		UnimplementedKeyLeakServiceServer: hydrav1.UnimplementedKeyLeakServiceServer{},
		db:                                cfg.DB,
		members:                           cfg.Members,
		email:                             cfg.Email,
		slack:                             slack.NewClient(),
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
