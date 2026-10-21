package domainverify

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// revokeDB records the writes RevokeContestedClaim makes. The embedded
// interface is nil, so an unexpected query panics.
type revokeDB struct {
	db.Querier
	claim  *db.FindVerifiedDomainClaimExcludingWorkspaceRow
	events []string
}

func (f *revokeDB) FindVerifiedDomainClaimExcludingWorkspace(_ context.Context, _ db.FindVerifiedDomainClaimExcludingWorkspaceParams) (db.FindVerifiedDomainClaimExcludingWorkspaceRow, error) {
	if f.claim == nil {
		return db.FindVerifiedDomainClaimExcludingWorkspaceRow{}, sql.ErrNoRows //nolint:exhaustruct
	}
	return *f.claim, nil
}

func (f *revokeDB) UpdateCustomDomainFailed(_ context.Context, arg db.UpdateCustomDomainFailedParams) error {
	f.events = append(f.events, "custom failed "+arg.ID+": "+arg.VerificationError.String)
	return nil
}

func (f *revokeDB) UpdatePortalDomainFailed(_ context.Context, arg db.UpdatePortalDomainFailedParams) error {
	f.events = append(f.events, "portal failed "+arg.ID+": "+arg.VerificationError.String)
	return nil
}

func (f *revokeDB) DeleteFrontlineRouteByFQDN(_ context.Context, fqdn string) error {
	f.events = append(f.events, "delete route "+fqdn)
	return nil
}

func (f *revokeDB) DeleteAcmeChallengeByDomainID(_ context.Context, domainID string) error {
	f.events = append(f.events, "delete acme "+domainID)
	return nil
}

func TestRevokeContestedClaimDispatchesOnSource(t *testing.T) {
	cases := map[string]struct {
		claim db.FindVerifiedDomainClaimExcludingWorkspaceRow
		want  []string
	}{
		"custom": {
			claim: db.FindVerifiedDomainClaimExcludingWorkspaceRow{Source: "custom", ID: "dom_old", WorkspaceID: "ws_old"},
			want: []string{
				"custom failed dom_old: domain claimed by another workspace",
				"delete route api.example.com",
				"delete acme dom_old",
			},
		},
		"portal": {
			claim: db.FindVerifiedDomainClaimExcludingWorkspaceRow{Source: "portal", ID: "pdom_old", WorkspaceID: "ws_old"},
			want: []string{
				"portal failed pdom_old: domain claimed by another workspace",
				"delete route api.example.com",
				"delete acme pdom_old",
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := &revokeDB{Querier: nil, claim: &tc.claim, events: nil}
			require.NoError(t, RevokeContestedClaim(context.Background(), f, "api.example.com", "ws_new", 1))
			require.Equal(t, tc.want, f.events)
		})
	}
}

func TestRevokeContestedClaimWithoutClaimWritesNothing(t *testing.T) {
	f := &revokeDB{Querier: nil, claim: nil, events: nil}
	require.NoError(t, RevokeContestedClaim(context.Background(), f, "api.example.com", "ws_new", 1))
	require.Empty(t, f.events)
}

// A source the query does not produce means the query and this switch drifted
// apart; revoking nothing would leave two verified claims, so it must fail.
func TestRevokeContestedClaimRejectsUnknownSource(t *testing.T) {
	f := &revokeDB{Querier: nil, claim: &db.FindVerifiedDomainClaimExcludingWorkspaceRow{Source: "other", ID: "x_old", WorkspaceID: "ws_old"}, events: nil}
	require.Error(t, RevokeContestedClaim(context.Background(), f, "api.example.com", "ws_new", 1))
	require.Empty(t, f.events)
}
