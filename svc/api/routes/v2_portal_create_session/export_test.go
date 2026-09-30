package handler

import (
	"context"
	"database/sql"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/zen"
)

// MintSessionForTest mints from a caller-supplied portal row, so a test can
// hand it the row a lagging replica would have returned.
func (h *Handler) MintSessionForTest(ctx context.Context, s *zen.Session, portal db.Portal, externalID string) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	_, _, err = h.mintSession(ctx, s, principal, mintRequest{
		Portal:      portal,
		ExternalID:  externalID,
		Scopes:      []string{"keys:read"},
		KeyspaceIDs: []string{},
		ScopesJSON:  []byte(`{"keyspaceIds":[],"scopes":["keys:read"]}`),
		ReturnURL:   sql.NullString{Valid: false, String: ""},
	})
	return err
}
