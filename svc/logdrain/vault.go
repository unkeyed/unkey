package logdrain

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
)

func newVaultClient(cfg VaultConfig) vaultv1connect.VaultServiceClient {
	return vaultv1connect.NewVaultServiceClient(http.DefaultClient, cfg.URL,
		connect.WithInterceptors(connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
			return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				if (cfg.Token == "") == (cfg.TokenFile == "") {
					return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("exactly one vault token source is required"))
				}
				token := cfg.Token
				if cfg.TokenFile != "" {
					var err error
					token, err = readVaultToken(cfg.TokenFile)
					if err != nil {
						return nil, connect.NewError(connect.CodeUnauthenticated, err)
					}
				}
				req.Header().Set("Authorization", "Bearer "+token)
				return next(ctx, req)
			}
		})),
	)
}

func readVaultToken(path string) (token string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open vault token file: %w", err)
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	const sizeMax = 16 * 1024
	data, err := io.ReadAll(io.LimitReader(file, sizeMax+1))
	if err != nil {
		return "", fmt.Errorf("read vault token file: %w", err)
	}
	token = strings.TrimSpace(string(data))
	if token == "" || len(data) > sizeMax || strings.ContainsAny(token, " \t\r\n") {
		return "", errors.New("vault token file is empty or invalid")
	}
	return token, nil
}
