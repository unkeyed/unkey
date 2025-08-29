package acme

import (
	"context"
	"database/sql"
	"errors"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
)

func (s *Service) HandleCertificateVerification(
	ctx context.Context,
	req *connect.Request[ctrlv1.HandleCertificateVerificationRequest],
) (*connect.Response[ctrlv1.HandleCertificateVerificationResponse], error) {
	res := connect.NewResponse(&ctrlv1.HandleCertificateVerificationResponse{})

	domain, err := db.Query.FindDomainByDomain(ctx, s.db.RO(), req.Msg.GetDomain())
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}

		return nil, connect.NewError(connect.CodeInternal, err)
	}

	challenge, err := db.Query.FindDomainChallengeByToken(ctx, s.db.RO(), db.FindDomainChallengeByTokenParams{
		WorkspaceID: domain.WorkspaceID,
		DomainID:    domain.ID,
		Token:       sql.NullString{Valid: true, String: req.Msg.GetToken()},
	})
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}

		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if !challenge.Authorization.Valid {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("challenge hasn't been issued yet"))
	}

	res.Msg.Token = challenge.Authorization.String
	return res, nil
}
