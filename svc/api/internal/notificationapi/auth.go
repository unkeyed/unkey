package notificationapi

import (
	"strings"

	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
)

func Authorize(s *zen.Session, action permissions.Action, maskUnreadable bool) (*principal.Principal, error) {
	p, err := s.GetPrincipal()
	if err != nil {
		return nil, err
	}
	if err = p.Authorize(rbac.U(urn.New().Workspace(p.AuthorizedWorkspaceID).Notifications(), action)); err != nil {
		if maskUnreadable {
			err = apierrors.MaskInsufficientPermissionsAsNotFound(err, codes.Data.Notification.NotFound.URN(), "Notification resource not found.")
		}
		return nil, err
	}
	return p, nil
}

func PersonalUser(s *zen.Session) (*principal.Principal, error) {
	p, err := s.GetPrincipal()
	if err != nil {
		return nil, err
	}
	if p.Subject.Type == principal.SubjectTypeUser && strings.TrimSpace(p.Subject.ID) != "" {
		switch source := p.Source.(type) {
		case principal.JWTSource:
			if len(source.Roles) > 0 {
				return p, nil
			}
		case principal.CLISessionSource:
			return p, nil
		}
	}
	return nil, fault.New("personal notifications require a workspace user session",
		fault.Code(codes.Auth.Authorization.Forbidden.URN()),
		fault.Public("A workspace user session is required to manage personal notification preferences."),
	)
}
