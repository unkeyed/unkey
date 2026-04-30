package coordinator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/defaults"
	"github.com/unkeyed/unkey/svc/logdrain/internal/creds"
	"github.com/unkeyed/unkey/svc/logdrain/internal/sinks"
	"github.com/unkeyed/unkey/svc/logdrain/internal/sinks/axiom"
	"github.com/unkeyed/unkey/svc/logdrain/internal/sinks/datadog"
	"github.com/unkeyed/unkey/svc/logdrain/internal/sinks/otlp"
)

// Factory builds a Sink from a drain row plus its decrypted credential.
// Stateless aside from the shared http.Client, which is held here so that
// every sink the coordinator constructs reuses the same connection pool.
//
// Per-provider unmarshal + construction lives in the providerBuilders
// registry below: each entry takes the row's raw config JSON, the
// decrypted token, and the shared client, and returns a constructed
// Sink. Adding a new provider is one entry plus the matching enum value
// in the schema — no switch statement to update, no shared "drainConfig"
// struct that grows a column per provider.
type Factory struct {
	creds      *creds.Cache
	httpClient *http.Client
}

func NewFactory(c *creds.Cache, client *http.Client) *Factory {
	return &Factory{
		creds:      c,
		httpClient: defaults.Or(client, http.DefaultClient),
	}
}

// ErrOAuthNotImplemented is returned when a row's credential_source is
// 'oauth' but the OAuth resolution path has not been wired up. v1 ships
// paste-token sinks; the OAuth path lands in the integrations stack.
var ErrOAuthNotImplemented = errors.New("coordinator: OAuth credential resolution not yet implemented")

// providerBuilder turns a row's raw config blob plus a decrypted token
// into a constructed Sink. Each provider implements one. Per-provider
// config types (axiom.Config, datadog.Config, otlp.Config) own their
// own JSON tags so the wire format and the Go type stay in one place.
type providerBuilder func(rawConfig json.RawMessage, token string, client *http.Client) (sinks.Sink, error)

var providerBuilders = map[db.LogDrainsProvider]providerBuilder{
	db.LogDrainsProviderAxiom: func(raw json.RawMessage, token string, client *http.Client) (sinks.Sink, error) {
		var cfg axiom.Config
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("parse axiom config: %w", err)
		}
		return axiom.New(cfg, token, client), nil
	},
	db.LogDrainsProviderDatadog: func(raw json.RawMessage, token string, client *http.Client) (sinks.Sink, error) {
		var cfg datadog.Config
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("parse datadog config: %w", err)
		}
		return datadog.New(cfg, token, client), nil
	},
	db.LogDrainsProviderOtlpHttp: func(raw json.RawMessage, _ string, client *http.Client) (sinks.Sink, error) {
		// OTLP carries auth inside cfg.AuthHeader (full header line);
		// the per-drain Vault-decrypted token is unused on this path.
		var cfg otlp.Config
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("parse otlp config: %w", err)
		}
		return otlp.New(cfg, client), nil
	},
}

// BuildSink constructs the right Sink for this drain row, decrypting the
// credential through the shared cache. Returns ErrOAuthNotImplemented
// when the drain references an oauth_grants row — the coordinator drops
// such drains until the OAuth path lands so we never silently send to
// the provider with a bogus token.
func (f *Factory) BuildSink(ctx context.Context, row db.ListEnabledLogDrainsRow) (sinks.Sink, error) {
	if !row.CredentialSource.Valid {
		return nil, fmt.Errorf("drain %s has no credential row", row.ID)
	}
	switch row.CredentialSource.LogDrainCredentialsSource {
	case db.LogDrainCredentialsSourcePaste:
		// proceed to decrypt below
	case db.LogDrainCredentialsSourceOauth:
		return nil, ErrOAuthNotImplemented
	default:
		return nil, fmt.Errorf("drain %s has unknown credential source", row.ID)
	}

	token, err := f.creds.Get(ctx, creds.Lookup{
		DrainID:     row.ID,
		WorkspaceID: row.WorkspaceID,
		Ciphertext:  row.EncryptedCredentials.String,
	})
	if err != nil {
		return nil, fmt.Errorf("decrypt drain %s: %w", row.ID, err)
	}

	build, ok := providerBuilders[row.Provider]
	if !ok {
		return nil, fmt.Errorf("drain %s has unknown provider %q", row.ID, row.Provider)
	}
	return build(row.Config, token, f.httpClient)
}
