package sessionauth

import (
	"context"
)

type localService struct {
	workspaceID string
}

// NewLocal creates a session auth service for local development. It accepts
// any token and always returns a fixed workspace ID. This mirrors the
// dashboard's LocalAuthProvider pattern.
func NewLocal(workspaceID string) Service {
	if workspaceID == "" {
		workspaceID = "ws_local_default"
	}
	return &localService{workspaceID: workspaceID}
}

func (s *localService) CanHandle(_ string) bool {
	return true
}

func (s *localService) Authenticate(_ context.Context, _ string) (*SessionResult, error) {
	return &SessionResult{
		WorkspaceID: s.workspaceID,
		UserID:      "user_local_admin",
		OrgID:       "org_localdefault",
		Role:        "admin",
		Permissions: []string{},
	}, nil
}
