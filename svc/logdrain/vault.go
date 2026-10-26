package logdrain

import (
	"net/http"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
	"github.com/unkeyed/unkey/pkg/rpc/interceptor"
)

func newVaultClient(cfg VaultConfig) vaultv1connect.VaultServiceClient {
	return vaultv1connect.NewVaultServiceClient(http.DefaultClient, cfg.URL,
		connect.WithInterceptors(interceptor.NewBearerAuth(cfg.Token, cfg.TokenFile)),
	)
}
