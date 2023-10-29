package keys

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/events"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutil"
)

func Test_SoftDeleteKey_DeletionTimeIsSet(t *testing.T) {

	resources := testutil.SetupResources(t)

	svc := New(Config{
		Database: resources.Database,
		Events:   events.NewNoop(),
	})

	ctx := context.Background()

	res, err := svc.CreateKey(ctx, &authenticationv1.CreateKeyRequest{
		WorkspaceId: resources.UserApi.WorkspaceId,
		KeyAuthId:   resources.UserApi.GetKeyAuthId(),
	})
	require.NoError(t, err)
	require.NotEmpty(t, res.Key)
	require.NotEmpty(t, res.KeyId)

	key, found, err := resources.Database.FindKeyById(ctx, res.KeyId)
	require.NoError(t, err)
	require.True(t, found)

	require.Nil(t, key.DeletedAt)

	_, err = svc.SoftDeleteKey(ctx, &authenticationv1.SoftDeleteKeyRequest{
		KeyId: res.KeyId,
	})
	require.NoError(t, err)

	key, found, err = resources.Database.FindKeyById(ctx, res.KeyId)
	require.NoError(t, err)
	require.True(t, found)
	require.NotNil(t, key.DeletedAt)
	require.True(t, time.UnixMilli(*key.DeletedAt).After(time.Now().Add(-time.Minute)))
	require.True(t, time.UnixMilli(*key.DeletedAt).Before(time.Now()))

}

type eventsMock struct {
	counter atomic.Int64
}

func (m *eventsMock) EmitKeyEvent(ctx context.Context, event events.KeyEvent) {
	m.counter.Add(1)

}
func (m *eventsMock) OnKeyEvent(func(ctx context.Context, event events.KeyEvent) error) {}

func Test_SoftDeleteKey_EmitsEvent(t *testing.T) {

	resources := testutil.SetupResources(t)

	e := &eventsMock{}

	svc := New(Config{
		Database: resources.Database,
		Events:   e,
		KeyCache: cache.NewNoopCache[*authenticationv1.Key](),
	})

	ctx := context.Background()

	res, err := svc.CreateKey(ctx, &authenticationv1.CreateKeyRequest{
		WorkspaceId: resources.UserApi.WorkspaceId,
		KeyAuthId:   resources.UserApi.GetKeyAuthId(),
	})
	require.NoError(t, err)
	require.NotEmpty(t, res.Key)
	require.NotEmpty(t, res.KeyId)

	key, found, err := resources.Database.FindKeyById(ctx, res.KeyId)
	require.NoError(t, err)
	require.True(t, found)

	require.Nil(t, key.DeletedAt)

	_, err = svc.SoftDeleteKey(ctx, &authenticationv1.SoftDeleteKeyRequest{
		KeyId: res.KeyId,
	})
	require.NoError(t, err)

	// 2 events are emitted, one for the key creation and one for the deletion
	require.Equal(t, int64(2), e.counter.Load())
}
