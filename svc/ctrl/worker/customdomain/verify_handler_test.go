package customdomain

import (
	"context"
	"database/sql"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
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

func mockStartedAt(mockCtx *mocks.MockContext, startedAt time.Time) {
	mockCtx.EXPECT().
		Run(mock.Anything, mock.AnythingOfType("*time.Time"), mock.Anything).
		Call.
		Run(func(args mock.Arguments) {
			*args.Get(1).(*time.Time) = startedAt
		}).
		Return(nil)
}
