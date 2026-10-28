package vaultreencryption

import (
	"context"
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

type pageResult struct {
	Cursor cursor
	Failed int32
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

// Only counters and cursors enter the journal, never plaintext or ciphertext.
func (h *Handler) Handle(ctx restate.ObjectContext, _ *hydrav1.RunVaultReEncryptionRequest) (*hydrav1.RunVaultReEncryptionResponse, error) {
	if h.vault == nil {
		return nil, restate.ToTerminalError(fmt.Errorf("Vault is not configured"))
	}
	var failed int32
	for _, table := range []struct {
		name string
		scan func(context.Context, cursor) pageResult
	}{
		{name: "encrypted_keys", scan: h.encryptedKeys},
		{name: "app_environment_variables", scan: h.environmentVariables},
		{name: "deployments", scan: h.deployments},
		{name: "logdrains", scan: h.logdrains},
		{name: "certificates", scan: h.certificates},
		{name: "shared_secrets", scan: h.sharedSecrets},
		{name: "clickhouse_workspace_settings", scan: h.clickhouseSettings},
		{name: "acme_users", scan: h.acmeUsers},
	} {
		after, err := restate.Get[cursor](ctx, table.name)
		if err != nil {
			return nil, err
		}
		for {
			page, err := restate.Run(ctx, func(rc restate.RunContext) (pageResult, error) {
				pageCtx, cancel := context.WithTimeout(rc, 30*time.Second)
				defer cancel()
				return table.scan(pageCtx, after), nil
			}, restate.WithName(fmt.Sprintf("re-encrypt %s after %d", table.name, after.Pk)))
			if err != nil {
				return nil, err
			}
			restate.Set(ctx, table.name, page.Cursor)
			failed += page.Failed
			if page.Cursor.Pk == 0 || page.Cursor == after {
				break
			}
			after = page.Cursor
		}
	}
	if failed > 0 {
		return nil, restate.ToTerminalError(fmt.Errorf("ciphertext re-encryption had %d failures; see table and pk in worker logs", failed))
	}
	if err := restate.RunVoid(ctx, func(rc restate.RunContext) error { return h.heartbeat.Ping(rc) }, restate.WithName("send heartbeat")); err != nil {
		return nil, err
	}
	return &hydrav1.RunVaultReEncryptionResponse{}, nil
}
