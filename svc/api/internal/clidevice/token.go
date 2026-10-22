package clidevice

import (
	"context"

	"github.com/unkeyed/unkey/pkg/auth/clisession"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/auth/workos"
	"github.com/unkeyed/unkey/pkg/hash"
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
	"github.com/unkeyed/unkey/pkg/uid"
	principalpermissions "github.com/unkeyed/unkey/svc/api/internal/principal"
)

func newRefreshToken() string {
	return "ukr_" + uid.Secure(32)
}

func hashRefreshToken(token string) string {
	return hash.Sha256(token)
}

func (s *Service) signAccessToken(userID string, name string, workspaceID string, sessionID string, device string, permissions []string) (string, error) {
	now := s.Clock.Now()
	return s.Signer.Sign(clisession.Claims{
		RegisteredClaims: tokenjwt.RegisteredClaims{
			Issuer:    clisession.Issuer,
			Subject:   userID,
			Audience:  []string{clisession.Audience},
			ExpiresAt: now.Add(accessTTL).Unix(),
			NotBefore: 0,
			IssuedAt:  now.Unix(),
			ID:        "",
		},
		WorkspaceID: workspaceID,
		SessionID:   sessionID,
		Permissions: permissions,
		TokenUse:    clisession.TokenUse,
		Name:        name,
		Device:      device,
	})
}

func intersectPermissions(ctx context.Context, workspaceID string, roles []string, stored []string) ([]string, error) {
	var subject principal.Subject
	caller := &principal.Principal{
		Version:               principal.Version,
		Type:                  principal.TypeJWT,
		Subject:               subject,
		Source:                nil,
		AuthorizedWorkspaceID: workspaceID,
		Permissions:           workos.PermissionsForRoles(workspaceID, roles),
	}
	kept := make([]string, 0, len(stored))
	for _, permission := range stored {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		validated, err := principalpermissions.ValidateDelegatedPermissions(ctx, caller, []string{permission})
		if err != nil {
			continue
		}
		kept = append(kept, validated...)
	}
	return kept, nil
}
