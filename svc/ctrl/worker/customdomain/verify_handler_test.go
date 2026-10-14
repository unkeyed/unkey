package customdomain

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/x/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// missingRowDB answers the domain lookup with NotFound. The embedded interface
// is nil, so any other query panics, pinning that VerifyDomain returns before
// touching anything else.
type missingRowDB struct{ db.Database }

func (missingRowDB) FindCustomDomainById(_ context.Context, _ string) (db.CustomDomain, error) {
	return db.CustomDomain{}, sql.ErrNoRows
}

// AddCustomDomain submits the workflow inside the transaction that inserts the
// domain row, so the first attempts can run before the commit lands. Within
// rowVisibilityGrace a missing row must surface as a retryable error, not a
// terminal one: terminal here would kill the workflow for good and strand the
// row in `pending` once the commit does land.
func TestVerifyDomainToleratesRowNotYetVisible(t *testing.T) {
	svc := New(Config{DB: missingRowDB{}, CnameDomain: "cname.unkey.local"})

	mockCtx := mocks.NewMockContext(t)
	mockCtx.EXPECT().Key().Return("dom_notyetvisible")
	mockStartedAt(mockCtx, time.Now())

	_, err := svc.VerifyDomain(restate.WithMockContext(mockCtx), &hydrav1.VerifyDomainRequest{})
	require.Error(t, err)
	require.False(t, restate.IsTerminalError(err),
		"a missing row inside the grace window must stay retryable, got: %v", err)
}

// Past the grace window a missing row can only mean deletion (DeleteCustomDomain,
// environment cascade, or a create whose commit failed after the submit), and the
// workflow must terminate instead of retrying against nothing for 24 hours. The
// window is exclusive: an attempt landing exactly on the boundary already
// terminates, pinning the `<` so the window cannot quietly become one retry wider.
func TestVerifyDomainTerminatesWhenRowStaysMissing(t *testing.T) {
	for name, age := range map[string]time.Duration{
		"past the window":         rowVisibilityGrace + time.Second,
		"exactly on the boundary": rowVisibilityGrace,
	} {
		t.Run(name, func(t *testing.T) {
			svc := New(Config{DB: missingRowDB{}, CnameDomain: "cname.unkey.local"})

			mockCtx := mocks.NewMockContext(t)
			mockCtx.EXPECT().Key().Return("dom_staysmissing")
			mockStartedAt(mockCtx, time.Now().Add(-age))

			_, err := svc.VerifyDomain(restate.WithMockContext(mockCtx), &hydrav1.VerifyDomainRequest{})
			require.Error(t, err)
			require.True(t, restate.IsTerminalError(err),
				"a row still missing after the grace window means deletion and must terminate, got: %v", err)
		})
	}
}

var errReachedStatusUpdate = errors.New("reached the verifying status update")

type staleRowDB struct {
	db.Database
	row db.CustomDomain
}

func (f staleRowDB) FindCustomDomainById(_ context.Context, _ string) (db.CustomDomain, error) {
	return f.row, nil
}

func (staleRowDB) UpdateCustomDomainVerificationStatus(_ context.Context, _ db.UpdateCustomDomainVerificationStatusParams) error {
	return errReachedStatusUpdate
}

// The deadline reads the journaled invocation start, not the row's created_at.
// This row is two windows old and the invocation is new, so the check must not
// trip. An implementation that reads created_at fails every retry of a domain
// older than 24 hours on its first attempt, before one DNS lookup runs.
func TestVerifyDomainOldRowDoesNotTimeOut(t *testing.T) {
	domainID := uid.New(uid.DomainPrefix)
	svc := New(Config{DB: staleRowDB{row: db.CustomDomain{
		ID:        domainID,
		Domain:    "retry.example.com",
		CreatedAt: time.Now().Add(-2 * maxVerificationDuration).UnixMilli(),
	}}, CnameDomain: "cname.unkey.local"})

	mockCtx := mocks.NewMockContext(t)
	mockCtx.EXPECT().Key().Return(domainID)
	mockStartedAt(mockCtx, time.Now())

	_, err := svc.VerifyDomain(restate.WithMockContext(mockCtx), &hydrav1.VerifyDomainRequest{})
	require.ErrorIs(t, err, errReachedStatusUpdate,
		"an old row on a fresh invocation must pass the deadline check, got: %v", err)
}

// The mirror of the test above. Here the row is new and the invocation is past
// the window. The two tests disagree about which clock is old, so together they
// pin which clock the deadline reads.
func TestVerifyDomainOldInvocationTimesOut(t *testing.T) {
	domainID := uid.New(uid.DomainPrefix)
	svc := New(Config{DB: staleRowDB{row: db.CustomDomain{
		ID:        domainID,
		Domain:    "timeout.example.com",
		CreatedAt: time.Now().UnixMilli(),
	}}, CnameDomain: "cname.unkey.local"})

	mockCtx := mocks.NewMockContext(t)
	mockCtx.EXPECT().Key().Return(domainID)
	mockStartedAt(mockCtx, time.Now().Add(-maxVerificationDuration-time.Second))
	// The mark-failed step writes into an *encoding.Void target, so it needs its
	// own expectation next to the *time.Time one above.
	mockCtx.EXPECT().Run(mock.Anything, mock.AnythingOfType("*encoding.Void"), mock.Anything).Return(nil)

	_, err := svc.VerifyDomain(restate.WithMockContext(mockCtx), &hydrav1.VerifyDomainRequest{})
	require.True(t, restate.IsTerminalError(err),
		"an invocation past the verification window must terminate, got: %v", err)
	require.ErrorContains(t, err, "timed out")
}

func mockStartedAt(mockCtx *mocks.MockContext, startedAt time.Time) {
	mockCtx.EXPECT().
		Run(mock.Anything, mock.AnythingOfType("*time.Time"), mock.Anything).
		Call.
		Run(func(args mock.Arguments) {
			*args.Get(1).(*time.Time) = startedAt
		}).
		Return(nil)
}

// frontlineRouteTable models frontline_routes as an in-memory table that holds
// one row per fully qualified domain name, so an insert for a name that still
// has a row fails the way MySQL rejects a duplicate key.
type frontlineRouteTable struct {
	db.Database
	routes map[string]db.InsertFrontlineRouteParams
	calls  []string
}

func (f *frontlineRouteTable) FindAppById(_ context.Context, _ string) (db.App, error) {
	return db.App{ID: "app_1", CurrentDeploymentID: sql.NullString{Valid: true, String: "d_1"}}, nil
}

func (f *frontlineRouteTable) DeleteFrontlineRouteByFQDN(_ context.Context, fqdn string) error {
	f.calls = append(f.calls, "delete "+fqdn)
	delete(f.routes, fqdn)
	return nil
}

func (f *frontlineRouteTable) InsertFrontlineRoute(_ context.Context, arg db.InsertFrontlineRouteParams) error {
	f.calls = append(f.calls, "insert "+arg.FullyQualifiedDomainName)
	if _, ok := f.routes[arg.FullyQualifiedDomainName]; ok {
		return fmt.Errorf("Duplicate entry '%s' for key 'frontline_routes.frontline_routes_fully_qualified_domain_name_unique' (errno 1062)", arg.FullyQualifiedDomainName)
	}
	f.routes[arg.FullyQualifiedDomainName] = arg
	return nil
}

// A route row for the verified name is the state a replayed step meets: the
// verify workflow retries the step for up to 24 hours, so the step must
// replace the row and land the route on its own app instead of colliding on
// the unique FQDN key on every attempt.
func TestCreateFrontlineRouteReplacesRowHoldingTheName(t *testing.T) {
	table := &frontlineRouteTable{routes: map[string]db.InsertFrontlineRouteParams{
		"app.example.com": {
			ID:                       "fr_previous",
			ProjectID:                "p_1",
			AppID:                    "app_1",
			FullyQualifiedDomainName: "app.example.com",
		},
	}}
	svc := New(Config{DB: table, CnameDomain: "cname.unkey.local"})

	require.NoError(t, svc.createFrontlineRoute(context.Background(), db.CustomDomain{
		ID:            "dom_1",
		WorkspaceID:   "ws_1",
		ProjectID:     "p_1",
		AppID:         "app_1",
		EnvironmentID: "env_1",
		Domain:        "app.example.com",
	}, time.Now().UnixMilli()))

	require.Equal(t, []string{"delete app.example.com", "insert app.example.com"}, table.calls)
	require.Len(t, table.routes, 1)
	require.Equal(t, "d_1", table.routes["app.example.com"].DeploymentID)
	require.Equal(t, db.FrontlineRoutesStickyLive, table.routes["app.example.com"].Sticky)
}
