package cluster

import (
	"context"
	"database/sql"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/cdc"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestWatchDeploymentChanges_StreamsStateAndCheckpoint(t *testing.T) {
	for _, test := range []struct {
		name       string
		lookupErr  error
		streamErr  error
		authorized bool
		replay     bool
		wantState  bool
		wantCode   connect.Code
	}{
		{name: "running state", authorized: true, wantState: true},
		{name: "replay discards token", authorized: true, replay: true, wantState: true},
		{name: "removed topology advances checkpoint", authorized: true, lookupErr: sql.ErrNoRows},
		{name: "transient lookup aborts without checkpoint", authorized: true, lookupErr: errors.New("database unavailable"), wantCode: connect.CodeInternal},
		{name: "expired position requires snapshot", authorized: true, streamErr: cdc.ErrExpired, wantCode: connect.CodeOutOfRange},
		{name: "invalid token", authorized: true, streamErr: cdc.ErrInvalidToken, wantCode: connect.CodeInvalidArgument},
		{name: "canceled stream", authorized: true, streamErr: context.Canceled, wantCode: connect.CodeCanceled},
		{name: "stream deadline", authorized: true, streamErr: context.DeadlineExceeded, wantCode: connect.CodeDeadlineExceeded},
		{name: "unauthenticated", wantCode: connect.CodeUnauthenticated},
	} {
		t.Run(test.name, func(t *testing.T) {
			clusterCache, err := cache.New(cache.Config[clusterCacheKey, db.FindClusterRow]{Fresh: time.Minute, Stale: time.Minute, MaxSize: 1, Resource: "test_cluster", Clock: clock.New()})
			require.NoError(t, err)
			t.Cleanup(clusterCache.Close)
			svc := &Service{
				db: &watchDatabase{lookupErr: test.lookupErr}, bearer: "test-token", clusterCache: clusterCache,
				deploymentStream: watchSource(func(ctx context.Context, region string, token []byte, change func(string) error, checkpoint func([]byte) error) error {
					if region != "region_test" {
						return errors.New("incorrect region filter")
					}
					if test.replay {
						if len(token) != 0 {
							return errors.New("replay retained old token")
						}
					} else if string(token) != "previous" {
						return errors.New("resume token was lost")
					}
					if test.streamErr != nil {
						return test.streamErr
					}
					if err := change("deploy_test"); err != nil {
						return err
					}
					return checkpoint([]byte("next"))
				}),
			}
			_, handler := ctrlv1connect.NewClusterServiceHandler(svc)
			server := httptest.NewServer(handler)
			t.Cleanup(server.Close)
			client := ctrlv1connect.NewClusterServiceClient(server.Client(), server.URL)
			req := connect.NewRequest(&ctrlv1.WatchDeploymentChangesRequest{Cluster: &ctrlv1.ClusterKey{CellId: "cell", Platform: "local", Region: "local"}, ResumeToken: []byte("previous"), Replay: test.replay})
			if test.authorized {
				req.Header().Set("Authorization", "Bearer test-token")
			}
			stream, err := client.WatchDeploymentChanges(t.Context(), req)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, stream.Close()) })
			var events []*ctrlv1.DeploymentChangeEvent
			for stream.Receive() {
				events = append(events, stream.Msg())
			}
			if test.wantCode != 0 {
				require.Equal(t, test.wantCode, connect.CodeOf(stream.Err()))
				require.Empty(t, events)
				return
			}
			require.NoError(t, stream.Err())
			if test.wantState {
				require.Len(t, events, 2)
				require.Equal(t, "deploy_test", events[0].GetDeployment().GetApply().GetDeploymentId())
				require.Empty(t, events[0].GetResumeToken())
			} else {
				require.Len(t, events, 1)
			}
			require.Equal(t, []byte("next"), events[len(events)-1].GetResumeToken())
		})
	}
}

type watchSource func(context.Context, string, []byte, func(string) error, func([]byte) error) error

func (s watchSource) Watch(ctx context.Context, region string, token []byte, change func(string) error, checkpoint func([]byte) error) error {
	return s(ctx, region, token, change, checkpoint)
}

type watchDatabase struct {
	db.Database
	lookupErr error
}

func (s *watchDatabase) FindCluster(context.Context, db.FindClusterParams) (db.FindClusterRow, error) {
	return db.FindClusterRow{RegionID: "region_test"}, nil
}

func (s *watchDatabase) FindDeploymentTopologyByDeploymentAndRegion(context.Context, db.FindDeploymentTopologyByDeploymentAndRegionParams) (db.FindDeploymentTopologyByDeploymentAndRegionRow, error) {
	return db.FindDeploymentTopologyByDeploymentAndRegionRow{ID: "deploy_test", DesiredStatus: db.DeploymentTopologyDesiredStatusRunning}, s.lookupErr
}
