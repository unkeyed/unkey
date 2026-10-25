package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	notificationv1 "github.com/unkeyed/unkey/gen/proto/notification/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/workos"
)

// AudienceResolver returns user IDs from the supplied, verified workspace membership.
type AudienceResolver func(context.Context, string, []workos.Member) ([]string, error)

type Config struct {
	DB          db.Database
	Vault       vault.VaultServiceClient
	Directory   workos.Directory
	Audiences   map[notificationv1.Audience]AudienceResolver
	Definitions []notification.Definition
	Providers   []notification.Provider
	Defaults    map[notification.Channel]string
	Clock       clock.Clock
}

type Service struct {
	db          db.Database
	vault       vault.VaultServiceClient
	directory   workos.Directory
	audiences   map[notificationv1.Audience]AudienceResolver
	definitions map[string]notification.Definition
	providers   map[string]notification.Provider
	defaults    map[notification.Channel]string
	clock       clock.Clock
}

func New(cfg Config) (*Service, error) {
	if err := assert.All(assert.NotNil(cfg.DB), assert.NotNil(cfg.Vault), assert.NotNil(cfg.Clock)); err != nil {
		return nil, err
	}
	s := &Service{
		db:          cfg.DB,
		vault:       cfg.Vault,
		directory:   cfg.Directory,
		clock:       cfg.Clock,
		audiences:   make(map[notificationv1.Audience]AudienceResolver),
		definitions: make(map[string]notification.Definition),
		providers:   make(map[string]notification.Provider),
		defaults:    make(map[notification.Channel]string),
	}
	for _, definition := range cfg.Definitions {
		if definition == nil || definition.Type() == "" || s.definitions[definition.Type()] != nil {
			return nil, errors.New("invalid or duplicate notification definition")
		}
		s.definitions[definition.Type()] = definition
	}
	for _, provider := range cfg.Providers {
		if provider == nil || provider.Name() == "" || len(provider.Name()) > 64 || s.providers[provider.Name()] != nil {
			return nil, errors.New("invalid or duplicate notification provider")
		}
		switch provider.Channel() {
		case notification.Email, notification.Slack, notification.Discord, notification.Webhook:
		default:
			return nil, errors.New("unsupported notification channel")
		}
		s.providers[provider.Name()] = provider
	}
	for channel, name := range cfg.Defaults {
		provider, ok := s.providers[name]
		if !ok || provider.Channel() != channel {
			return nil, errors.New("default notification provider does not support channel")
		}
		s.defaults[channel] = name
	}
	for audience, resolver := range cfg.Audiences {
		if audience == notificationv1.Audience_AUDIENCE_UNSPECIFIED || resolver == nil {
			return nil, errors.New("invalid notification audience resolver")
		}
		s.audiences[audience] = resolver
	}
	return s, nil
}

type deliveryRef struct {
	WorkspaceID string `json:"workspaceId"`
	ID          string `json:"id"`
}

func (s *Service) Definition() restate.ServiceDefinition {
	return restate.NewObject(notification.ServiceName).
		Handler("Notify", restate.NewObjectHandler(s.Notify, retryPolicy()))
}

func (s *Service) DeliveryDefinition() restate.ServiceDefinition {
	return restate.NewObject(notification.DeliveryServiceName).
		Handler("Deliver", restate.NewObjectHandler(s.Deliver, restate.WithIngressPrivate(true), retryPolicy()))
}

func (s *Service) Notify(ctx restate.ObjectContext, event notification.Event) (string, error) {
	if err := event.Validate(); err != nil {
		return "", invalid(err.Error())
	}
	if restate.Key(ctx) != event.ID() {
		return "", invalid("notification key does not match event")
	}
	deliveries, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]deliveryRef, error) {
		return s.plan(runCtx, event)
	}, restate.WithName("freeze notification routing"))
	if err != nil {
		return "", err
	}
	for _, delivery := range deliveries {
		restate.Object[string](ctx, notification.DeliveryServiceName, delivery.ID, "Deliver").Send(delivery, restate.WithIdempotencyKey(delivery.ID))
	}
	return event.ID(), nil
}

func (s *Service) Deliver(ctx restate.ObjectContext, ref deliveryRef) (string, error) {
	if restate.Key(ctx) != ref.ID || ref.WorkspaceID == "" {
		return "", invalid("invalid delivery key")
	}
	for {
		result, err := restate.Run(ctx, func(runCtx restate.RunContext) (attemptResult, error) {
			return s.attempt(runCtx, ref)
		}, restate.WithName("attempt notification delivery"))
		if err != nil {
			return "", err
		}
		if result.RetryAfter <= 0 {
			return string(result.Status), nil
		}
		if err := restate.Sleep(ctx, result.RetryAfter); err != nil {
			return "", err
		}
	}
}

func retryPolicy() restate.HandlerOption {
	return restate.WithInvocationRetryPolicy(
		restate.WithInitialRetryInterval(time.Second),
		restate.WithMaxRetryInterval(time.Minute),
		restate.WithMaxRetryAttempts(20),
		restate.PauseOnMaxAttempts(),
	)
}

func invalid(reason string) error {
	return restate.ToTerminalError(fmt.Errorf("notification: %s", reason), restate.WithErrorCode(400))
}
