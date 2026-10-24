package customdomain

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// missingRowDB answers the domain lookup with NotFound. The embedded interface
// is nil, so any other query panics, pinning that checkDomain returns before
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

	_, err := svc.checkDomain(context.Background(), "dom_notyetvisible", time.Now())
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

			_, err := svc.checkDomain(context.Background(), "dom_staysmissing", time.Now().Add(-age))
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

	_, err := svc.checkDomain(context.Background(), domainID, time.Now())
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

	check, err := svc.checkDomain(context.Background(), domainID, time.Now().Add(-maxVerificationDuration-time.Second))
	require.NoError(t, err)
	require.True(t, check.TimedOut, "an invocation past the verification window must time out")
}
