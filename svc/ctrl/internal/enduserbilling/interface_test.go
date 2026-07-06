package enduserbilling

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakePusher records the requests it receives so dispatch tests can assert
// records arrive unchanged.
type fakePusher struct {
	requests []PushRequest
}

var _ MeterPusher = (*fakePusher)(nil)

func (f *fakePusher) Push(ctx context.Context, req PushRequest) (int, error) {
	f.requests = append(f.requests, req)
	return len(req.Records), nil
}

func TestNoopPushesNothing(t *testing.T) {
	n := NewNoop()
	pushed, err := n.Push(context.Background(), PushRequest{
		WorkspaceID: "ws_1",
		Year:        2026,
		Month:       6,
		Records: []UsageRecord{
			{
				IdentityID:         "id_1",
				ExternalID:         "user_1",
				ProviderCustomerID: "cus_1",
				RateCardID:         "rc_1",
				Verifications:      100,
				SpentCredits:       50,
				RatelimitsPassed:   10,
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, 0, pushed)
}

func TestFakeReceivesRecordsUnchanged(t *testing.T) {
	f := &fakePusher{requests: nil}
	req := PushRequest{
		WorkspaceID: "ws_1",
		Year:        2026,
		Month:       6,
		Records: []UsageRecord{
			{
				IdentityID:         "id_1",
				ExternalID:         "user_1",
				ProviderCustomerID: "cus_1",
				RateCardID:         "rc_1",
				Verifications:      100,
				SpentCredits:       50,
				RatelimitsPassed:   10,
			},
			{
				IdentityID:         "id_2",
				ExternalID:         "user_2",
				ProviderCustomerID: "cus_2",
				RateCardID:         "rc_2",
				Verifications:      1,
				SpentCredits:       0,
				RatelimitsPassed:   0,
			},
		},
	}
	pushed, err := f.Push(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, 2, pushed)
	require.Len(t, f.requests, 1)
	require.Equal(t, req, f.requests[0])
}

func TestUsageRecordPositive(t *testing.T) {
	require.False(t, UsageRecord{}.Positive())                   //nolint:exhaustruct
	require.True(t, UsageRecord{Verifications: 1}.Positive())    //nolint:exhaustruct
	require.True(t, UsageRecord{SpentCredits: 1}.Positive())     //nolint:exhaustruct
	require.True(t, UsageRecord{RatelimitsPassed: 1}.Positive()) //nolint:exhaustruct
}
