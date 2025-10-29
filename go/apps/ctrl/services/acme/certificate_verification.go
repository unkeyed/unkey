package acme

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
)

func (s *Service) HandleCertificateVerification(
	ctx context.Context,
	req *connect.Request[ctrlv1.HandleCertificateVerificationRequest],
) (*connect.Response[ctrlv1.HandleCertificateVerificationResponse], error) {
	res := connect.NewResponse(&ctrlv1.HandleCertificateVerificationResponse{Token: ""})

	domain, err := db.Query.FindDomainByDomain(ctx, s.db.RO(), req.Msg.GetDomain())
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}

		return nil, connect.NewError(connect.CodeInternal, err)
	}

	challenge, err := db.Query.FindAcmeChallengeByToken(ctx, s.db.RO(), db.FindAcmeChallengeByTokenParams{
		WorkspaceID: domain.WorkspaceID,
		DomainID:    domain.ID,
		Token:       req.Msg.GetToken(),
	})
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}

		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if challenge.Authorization == "" {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("challenge hasn't been issued yet"))
	}

	res.Msg.Token = challenge.Authorization
	return res, nil
}
