package certificate

import (
	"fmt"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/x/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestRenewalCountsInvocationHandlesAsSubmissions(t *testing.T) {
	ctx := mocks.NewMockContext(t)
	challenges := []db.ListExecutableChallengesRow{
		{Domain: "one.example.com", WorkspaceID: "ws_one"},
		{Domain: "two.example.com", WorkspaceID: "ws_two"},
	}
	expectRun(t, ctx, "list expiring certificates", challenges, nil).Once()
	for _, challenge := range challenges {
		client := mocks.NewMockClient(t)
		ctx.EXPECT().Object("hydra.v1.CertificateService", challenge.Domain, "ProcessChallenge", mock.Anything).Return(client).Once()
		invocation := client.EXPECT().MockSend(&hydrav1.ProcessChallengeRequest{
			WorkspaceId: challenge.WorkspaceID,
			Domain:      challenge.Domain,
		})
		invocation.EXPECT().GetInvocationId().Return("invocation-" + challenge.Domain).Once()
	}
	ctx.EXPECT().Sleep(100 * time.Millisecond).Return(nil).Twice()
	expectRun(t, ctx, "send heartbeat", restate.Void{}, nil).Once()

	svc := New(Config{})
	response, err := svc.RenewExpiringCertificates(restate.WithMockContext(ctx), &hydrav1.RenewExpiringCertificatesRequest{})
	require.NoError(t, err)
	require.Equal(t, int32(2), response.GetCertificatesChecked())
	require.Equal(t, int32(2), response.GetRenewalsTriggered())
}

func TestRenewalDoesNotHeartbeatAfterListFailure(t *testing.T) {
	ctx := mocks.NewMockContext(t)
	expected := restate.TerminalErrorf("database unavailable")
	expectRun(t, ctx, "list expiring certificates", []db.ListExecutableChallengesRow{}, expected).Once()

	svc := New(Config{})
	response, err := svc.RenewExpiringCertificates(restate.WithMockContext(ctx), &hydrav1.RenewExpiringCertificatesRequest{})
	require.ErrorIs(t, err, expected)
	require.Nil(t, response)
}

func expectRun[T any](t *testing.T, ctx *mocks.MockContext, name string, value T, err restate.TerminalError, extraOptions ...any) *mock.Call {
	t.Helper()
	options := append([]any{restate.WithName(name)}, extraOptions...)
	return ctx.EXPECT().Run(mock.Anything, mock.AnythingOfType(fmt.Sprintf("%T", new(T))), options...).
		Run(func(_ func(restate.RunContext) (any, error), output any, _ ...restate.RunOption) {
			*output.(*T) = value
		}).Return(err).Call
}
