package engine

import (
	"context"
	"fmt"
	"net/http"

	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/svc/logdrain/internal/db"
	"github.com/unkeyed/unkey/svc/logdrain/sink"
	"github.com/unkeyed/unkey/svc/logdrain/sink/axiom"
	"github.com/unkeyed/unkey/svc/logdrain/sink/httpdrain"
	"google.golang.org/protobuf/proto"
)

// factory builds one sink per delivery attempt from the drain row and its decrypted credentials.
type factory struct {
	vault vault.VaultServiceClient

	// decryptCache maps workspace-scoped ciphertext to plaintext to avoid one
	// Vault RPC per batch. The ciphertext key is safe: decryption is
	// deterministic, and a credentials update writes a new ciphertext, so
	// stale entries can never serve the wrong plaintext. The TTL only bounds
	// how long plaintext credentials stay in memory.
	decryptCache cache.Cache[cache.ScopedKey, string]

	// unsafeAllowPrivateEndpoints disables the HTTP sink SSRF guard. Development only.
	unsafeAllowPrivateEndpoints bool
}

// build creates the destination sink only after its stored configuration and credentials decode successfully.
func (f factory) build(ctx context.Context, drain db.GetLeasedAndDueLogdrainRow) (sink.Sink, error) {
	cfg := &logdrainv1.Config{}
	if err := proto.Unmarshal(drain.Config, cfg); err != nil {
		return nil, fmt.Errorf("decode logdrain config: %w", err)
	}
	switch destination := cfg.Destination.(type) {
	case *logdrainv1.Config_Http:
		return f.buildHTTP(ctx, drain.WorkspaceID, destination.Http)
	case *logdrainv1.Config_Axiom:
		token, err := f.decrypt(ctx, drain.WorkspaceID, destination.Axiom.GetEncryptedToken())
		if err != nil {
			return nil, err
		}
		built, err := axiom.New(axiom.Config{
			Dataset: destination.Axiom.GetDataset(),
			Token:   token,
			Timeout: deliveryTimeout,
		})
		if err != nil {
			return nil, fmt.Errorf("build axiom sink: %w", err)
		}
		return built, nil
	default:
		return nil, fmt.Errorf("logdrain provider is required")
	}
}

// buildHTTP supports optional headers while keeping all credentials encrypted at rest.
func (f factory) buildHTTP(ctx context.Context, workspaceID string, cfg *logdrainv1.HttpConfig) (sink.Sink, error) {
	headers := http.Header{}
	for _, header := range cfg.GetHeaders() {
		plaintext, err := f.decrypt(ctx, workspaceID, header.GetEncryptedValue())
		if err != nil {
			return nil, err
		}
		headers.Add(header.GetName(), plaintext)
	}
	built, err := httpdrain.New(httpdrain.Config{
		Endpoint:                cfg.GetUrl(),
		Format:                  cfg.GetFormat(),
		Headers:                 headers,
		Timeout:                 deliveryTimeout,
		UnsafeAllowTestEndpoint: f.unsafeAllowPrivateEndpoints,
	})
	if err != nil {
		return nil, fmt.Errorf("build http sink: %w", err)
	}
	return built, nil
}

// decrypt uses the workspace ID as the keyring so credentials remain isolated between workspaces.
func (f factory) decrypt(ctx context.Context, keyring, encrypted string) (string, error) {
	cacheKey := cache.ScopedKey{WorkspaceID: keyring, Key: encrypted}
	if plaintext, hit := f.decryptCache.Get(ctx, cacheKey); hit == cache.Hit {
		return plaintext, nil
	}
	resp, err := f.vault.Decrypt(ctx, &vaultv1.DecryptRequest{Keyring: keyring, Encrypted: encrypted})
	if err != nil {
		return "", fmt.Errorf("decrypt logdrain secret: %w", err)
	}
	f.decryptCache.Set(ctx, cacheKey, resp.GetPlaintext())
	return resp.GetPlaintext(), nil
}
