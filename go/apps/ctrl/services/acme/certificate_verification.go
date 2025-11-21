package acme

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
)

func (s *Service) VerifyCertificate(
	ctx context.Context,
	req *connect.Request[ctrlv1.VerifyCertificateRequest],
) (*connect.Response[ctrlv1.VerifyCertificateResponse], error) {
	res := connect.NewResponse(&ctrlv1.VerifyCertificateResponse{Authorization: ""})

	domain, err := db.Query.FindCustomDomainByDomain(ctx, s.db.RO(), req.Msg.GetDomain())
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

	res.Msg.Authorization = challenge.Authorization
	return res, nil
}
