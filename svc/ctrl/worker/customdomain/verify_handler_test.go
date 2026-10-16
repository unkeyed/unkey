package customdomain

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/x/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
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

func TestCreateFrontlineRouteReplacesRowHoldingTheName(t *testing.T) {
	ctx := t.Context()
	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	seeder := seed.New(t, database, nil)
	app := seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: uid.New(uid.WorkspacePrefix),
		ProjectID:   uid.New(uid.ProjectPrefix),
		Name:        "verified",
		Slug:        "verified",
	})
	deploymentID := uid.New(uid.DeploymentPrefix)
	now := time.Now().UnixMilli()
	require.NoError(t, database.UpdateAppDeployments(ctx, db.UpdateAppDeploymentsParams{
		AppID:               app.ID,
		CurrentDeploymentID: sql.NullString{Valid: true, String: deploymentID},
		UpdatedAt:           sql.NullInt64{Valid: true, Int64: now},
	}))
	dom := db.CustomDomain{
		ID:            uid.New(uid.DomainPrefix),
		WorkspaceID:   app.WorkspaceID,
		ProjectID:     app.ProjectID,
		AppID:         app.ID,
		EnvironmentID: uid.New(uid.EnvironmentPrefix),
		Domain:        uid.DNS1035() + ".example.com",
	}
	previous := db.InsertFrontlineRouteParams{
		ID:                       uid.New(uid.FrontlineRoutePrefix),
		ProjectID:                uid.New(uid.ProjectPrefix),
		AppID:                    uid.New(uid.AppPrefix),
		DeploymentID:             uid.New(uid.DeploymentPrefix),
		EnvironmentID:            uid.New(uid.EnvironmentPrefix),
		FullyQualifiedDomainName: dom.Domain,
		Sticky:                   db.FrontlineRoutesStickyEnvironment,
		CreatedAt:                now - 1,
	}
	require.NoError(t, database.InsertFrontlineRoute(ctx, previous))
	svc := New(Config{DB: database, CnameDomain: "cname.unkey.local"})

	t.Run("failed replacement preserves the previous route", func(t *testing.T) {
		before, err := database.FindFrontlineRouteByFQDN(ctx, dom.Domain)
		require.NoError(t, err)
		invalid := dom
		invalid.EnvironmentID = strings.Repeat("x", 49)
		require.Error(t, svc.createFrontlineRoute(ctx, invalid, now))
		after, err := database.FindFrontlineRouteByFQDN(ctx, dom.Domain)
		require.NoError(t, err)
		require.Equal(t, before, after)
	})

	require.NoError(t, svc.createFrontlineRoute(ctx, dom, now))
	route, err := database.FindFrontlineRouteByFQDN(ctx, dom.Domain)
	require.NoError(t, err)
	require.NotEqual(t, previous.ID, route.ID)
	require.Equal(t, dom.ProjectID, route.ProjectID)
	require.Equal(t, dom.AppID, route.AppID)
	require.Equal(t, dom.EnvironmentID, route.EnvironmentID)
	require.Equal(t, deploymentID, route.DeploymentID)
	require.Equal(t, db.FrontlineRoutesStickyLive, route.Sticky)

	require.NoError(t, svc.createFrontlineRoute(ctx, dom, now))
	replayed, err := database.FindFrontlineRouteByFQDN(ctx, dom.Domain)
	require.NoError(t, err)
	require.Equal(t, route, replayed)

	require.NoError(t, database.ReassignFrontlineRoute(ctx, db.ReassignFrontlineRouteParams{
		ID:           previous.ID,
		DeploymentID: previous.DeploymentID,
		UpdatedAt:    sql.NullInt64{Valid: true, Int64: now + 1},
	}))
	afterOldOwnerWrite, err := database.FindFrontlineRouteByFQDN(ctx, dom.Domain)
	require.NoError(t, err)
	require.Equal(t, route, afterOldOwnerWrite)
}
