package vaultreencryption

import (
	"context"
	"errors"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

const batchSize = 100

type Config struct {
	DB db.Database
	// Vault may be nil so workers without Vault can still run other cron jobs.
	Vault     vault.VaultServiceClient
	Heartbeat healthcheck.Heartbeat
}

type Handler struct {
	db        db.Database
	vault     vault.VaultServiceClient
	heartbeat healthcheck.Heartbeat
}

type cursor struct {
	Pk uint64
}

func New(cfg Config) (*Handler, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop()"),
	); err != nil {
		return nil, err
	}
	return &Handler{db: cfg.DB, vault: cfg.Vault, heartbeat: cfg.Heartbeat}, nil
}

// Re-encryption errors must not contain plaintext or ciphertext: Restate journals them.
func (h *Handler) Handle(ctx restate.ObjectContext, _ *hydrav1.RunVaultReEncryptionRequest) (*hydrav1.RunVaultReEncryptionResponse, error) {
	if h.vault == nil {
		return nil, restate.ToTerminalError(fmt.Errorf("Vault is not configured"))
	}
	var failures []error
	for _, table := range []struct {
		name          string
		reencryptPage func(context.Context, cursor) (cursor, error)
	}{
		{name: "encrypted_keys", reencryptPage: h.encryptedKeys},
		{name: "app_environment_variables", reencryptPage: h.environmentVariables},
		{name: "deployments", reencryptPage: h.deployments},
		{name: "logdrains", reencryptPage: h.logdrains},
		{name: "certificates", reencryptPage: h.certificates},
		{name: "shared_secrets", reencryptPage: h.sharedSecrets},
		{name: "clickhouse_workspace_settings", reencryptPage: h.clickhouseSettings},
		{name: "acme_users", reencryptPage: h.acmeUsers},
	} {
		after, err := restate.Get[cursor](ctx, table.name)
		if err != nil {
			return nil, err
		}
		for {
			next, err := restate.Run(ctx, func(rc restate.RunContext) (cursor, error) {
				pageCtx, cancel := context.WithTimeout(rc, 30*time.Second)
				defer cancel()
				next, err := table.reencryptPage(pageCtx, after)
				return next, restate.ToTerminalError(err)
			}, restate.WithName(fmt.Sprintf("re-encrypt %s after %d", table.name, after.Pk)))
			if err != nil {
				failures = append(failures, fmt.Errorf("%s: %s", table.name, err))
				break
			}
			restate.Set(ctx, table.name, next)
			if next.Pk == 0 {
				break
			}
			after = next
		}
	}
	if err := errors.Join(failures...); err != nil {
		return nil, restate.ToTerminalError(err)
	}
	if err := restate.RunVoid(ctx, func(rc restate.RunContext) error { return h.heartbeat.Ping(rc) }, restate.WithName("send heartbeat")); err != nil {
		return nil, err
	}
	return &hydrav1.RunVaultReEncryptionResponse{}, nil
}
