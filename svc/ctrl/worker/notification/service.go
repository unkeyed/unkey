package notification

import (
	"errors"
	"time"

	restate "github.com/restatedev/sdk-go"
	notificationv1 "github.com/unkeyed/unkey/gen/proto/notification/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/notificationrouting"
	"github.com/unkeyed/unkey/svc/ctrl/internal/workos"
)

// AudienceResolver returns user IDs from the supplied, verified workspace membership.
type AudienceResolver = notificationrouting.AudienceResolver

// Config selects event definitions, channel providers, and workspace audience resolvers.
// Directory is required only for email targets. Defaults maps each channel to a provider name.
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

// Service freezes notification routes and runs independently retried deliveries.
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

// New rejects duplicate definitions and providers and mismatched channel defaults.
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

// Definition binds notification planning to an object keyed by event identity.
func (s *Service) Definition() restate.ServiceDefinition {
	return restate.NewObject(notification.ServiceName).
		Handler("Notify", restate.NewObjectHandler(s.Notify, retryPolicy()))
}

// DeliveryDefinition binds private delivery attempts to an object keyed by delivery ID.
func (s *Service) DeliveryDefinition() restate.ServiceDefinition {
	return restate.NewObject(notification.DeliveryServiceName).
		Handler("Deliver", restate.NewObjectHandler(s.Deliver, restate.WithIngressPrivate(true), retryPolicy()))
}

func retryPolicy() restate.HandlerOption {
	return restate.WithInvocationRetryPolicy(
		restate.WithInitialRetryInterval(time.Second),
		restate.WithMaxRetryInterval(time.Minute),
		restate.WithMaxRetryAttempts(20),
		restate.PauseOnMaxAttempts(),
	)
}
