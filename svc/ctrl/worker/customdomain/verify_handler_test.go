package customdomain

import (
	"context"
	"database/sql"
	"errors"
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
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestVerificationCannotRecreateChallengeAfterDomainDeletion(t *testing.T) {
	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	svc := New(Config{DB: database})
	dom := db.CustomDomain{ID: uid.New(uid.DomainPrefix), WorkspaceID: uid.New(uid.WorkspacePrefix)}
	mockCtx := mocks.NewMockContext(t)
	mockCtx.EXPECT().Value(mock.Anything).Return(nil).Maybe()
	mockCtx.EXPECT().Deadline().Return(time.Time{}, false).Maybe()
	mockCtx.EXPECT().Done().Return(nil).Maybe()
	mockCtx.EXPECT().Err().Return(nil).Maybe()
	mockCtx.EXPECT().Run(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
		func(fn func(restate.RunContext) (any, error), _ any, _ ...restate.RunOption) restate.TerminalError {
			_, err := fn(mockCtx)
			if err != nil {
				require.True(t, restate.IsTerminalError(err), "%v", err)
				return restate.ToTerminalError(err)
			}
			return nil
		},
	).Twice()
	_, err = svc.onVerificationSuccess(restate.WithMockContext(mockCtx), dom)
	require.True(t, restate.IsTerminalError(err), "%v", err)
	var count int
	require.NoError(t, database.RW().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM acme_challenges WHERE domain_id = ?", dom.ID).Scan(&count))
	require.Zero(t, count)
}

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
