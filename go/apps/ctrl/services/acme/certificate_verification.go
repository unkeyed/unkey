package acme

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/db"
)

func (s *Service) VerifyCertificate(
	ctx context.Context,
	req *connect.Request[ctrlv1.VerifyCertificateRequest],
) (*connect.Response[ctrlv1.VerifyCertificateResponse], error) {
	res := connect.NewResponse(&ctrlv1.VerifyCertificateResponse{Authorization: ""})

	domainName := req.Msg.GetDomain()
	token := req.Msg.GetToken()

	// Look up domain with cache
	domain, hit, err := s.domainCache.SWR(ctx, domainName,
		func(ctx context.Context) (db.CustomDomain, error) {
			return db.Query.FindCustomDomainByDomain(ctx, s.db.RO(), domainName)
		},
		caches.DefaultFindFirstOp,
	)
	if err != nil && !db.IsNotFound(err) {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if hit == cache.Null || db.IsNotFound(err) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("domain not found"))
	}

	// Look up challenge with cache
	// Key format: domainID|token
	challengeKey := domain.ID + "|" + token
	challenge, hit, err := s.challengeCache.SWR(ctx, challengeKey,
		func(ctx context.Context) (db.AcmeChallenge, error) {
			return db.Query.FindAcmeChallengeByToken(ctx, s.db.RO(), db.FindAcmeChallengeByTokenParams{
				WorkspaceID: domain.WorkspaceID,
				DomainID:    domain.ID,
				Token:       token,
			})
		},
		caches.DefaultFindFirstOp,
	)
	if err != nil && !db.IsNotFound(err) {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if hit == cache.Null || db.IsNotFound(err) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("challenge not found"))
	}

	if challenge.Authorization == "" {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("challenge hasn't been issued yet"))
	}

	res.Msg.Authorization = challenge.Authorization
	return res, nil
}
