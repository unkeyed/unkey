package connect

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bufbuild/connect-go"
	vaultv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/agent/gen/proto/vault/v1/vaultv1connect"
	"github.com/unkeyed/unkey/apps/agent/pkg/auth"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"github.com/unkeyed/unkey/apps/agent/services/vault"
)

type vaultServer struct {
	svc    *vault.Service
	logger logging.Logger
	vaultv1connect.UnimplementedVaultServiceHandler
}

func NewVaultServer(svc *vault.Service, logger logging.Logger) *vaultServer {

	return &vaultServer{
		svc:    svc,
		logger: logger,
	}
}

func (s *vaultServer) CreateHandler() (string, http.Handler) {
	return vaultv1connect.NewVaultServiceHandler(s)

}

func (s *vaultServer) CreateDEK(
	ctx context.Context,
	req *connect.Request[vaultv1.CreateDEKRequest],
) (*connect.Response[vaultv1.CreateDEKResponse], error) {
	ctx, span := tracing.Start(ctx, tracing.NewSpanName("connect.vault", "CreateDEK"))
	defer span.End()
	authorization := req.Header().Get("Authorization")
	err := auth.Authorize(ctx, authorization)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to authorize request")
		return nil, err
	}

	res, err := s.svc.CreateDEK(ctx, req.Msg)
	if err != nil {
		return nil, fmt.Errorf("failed to create dek: %w", err)
	}
	return connect.NewResponse(res), nil

}

func (s *vaultServer) Decrypt(
	ctx context.Context,
	req *connect.Request[vaultv1.DecryptRequest],
) (*connect.Response[vaultv1.DecryptResponse], error) {
	ctx, span := tracing.Start(ctx, tracing.NewSpanName("connect.vault", "Decrypt"))
	defer span.End()
	authorization := req.Header().Get("Authorization")
	err := auth.Authorize(ctx, authorization)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to authorize request")
		return nil, err
	}

	res, err := s.svc.Decrypt(ctx, req.Msg)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	return connect.NewResponse(res), nil

}

func (s *vaultServer) Encrypt(
	ctx context.Context,
	req *connect.Request[vaultv1.EncryptRequest],
) (*connect.Response[vaultv1.EncryptResponse], error) {
	ctx, span := tracing.Start(ctx, tracing.NewSpanName("connect.vault", "Encrypt"))
	defer span.End()
	authorization := req.Header().Get("Authorization")
	err := auth.Authorize(ctx, authorization)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to authorize request")
		return nil, err
	}

	res, err := s.svc.Encrypt(ctx, req.Msg)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt: %w", err)
	}
	return connect.NewResponse(res), nil

}

func (s *vaultServer) EncryptBulk(
	ctx context.Context,
	req *connect.Request[vaultv1.EncryptBulkRequest],
) (*connect.Response[vaultv1.EncryptBulkResponse], error) {
	ctx, span := tracing.Start(ctx, tracing.NewSpanName("connect.vault", "EncryptBulk"))
	defer span.End()
	authorization := req.Header().Get("Authorization")
	err := auth.Authorize(ctx, authorization)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to authorize request")
		return nil, err
	}

	res, err := s.svc.EncryptBulk(ctx, req.Msg)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt bulk: %w", err)
	}
	return connect.NewResponse(res), nil

}

func (s *vaultServer) ReEncrypt(
	ctx context.Context,
	req *connect.Request[vaultv1.ReEncryptRequest],
) (*connect.Response[vaultv1.ReEncryptResponse], error) {
	ctx, span := tracing.Start(ctx, tracing.NewSpanName("connect.vault", "Reencrypt"))
	defer span.End()
	authorization := req.Header().Get("Authorization")
	err := auth.Authorize(ctx, authorization)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to authorize request")
		return nil, err
	}

	res, err := s.svc.ReEncrypt(ctx, req.Msg)
	if err != nil {
		return nil, fmt.Errorf("failed to reencrypt: %w", err)
	}
	return connect.NewResponse(res), nil

}

func (s *vaultServer) Liveness(
	ctx context.Context,
	req *connect.Request[vaultv1.LivenessRequest],
) (*connect.Response[vaultv1.LivenessResponse], error) {
	_, span := tracing.Start(ctx, tracing.NewSpanName("connect.vault", "Liveness"))
	defer span.End()

	return connect.NewResponse(&vaultv1.LivenessResponse{Status: "ok"}), nil

}
