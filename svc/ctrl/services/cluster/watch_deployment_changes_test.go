package cluster

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// stubDatabase implements db.Database for fetchDeploymentChangeEvents tests.
// Only the two methods the function calls are overridden; calling anything
// else panics via the embedded nil interface.
type stubDatabase struct {
	db.Database
	changes []db.DeploymentChange
	findRow db.FindDeploymentTopologyByDeploymentAndRegionRow
	findErr error
}

func (s *stubDatabase) ListDeploymentChangesByRegionAll(_ context.Context, _ db.ListDeploymentChangesByRegionAllParams) ([]db.DeploymentChange, error) {
	return s.changes, nil
}

func (s *stubDatabase) FindDeploymentTopologyByDeploymentAndRegion(_ context.Context, _ db.FindDeploymentTopologyByDeploymentAndRegionParams) (db.FindDeploymentTopologyByDeploymentAndRegionRow, error) {
	return s.findRow, s.findErr
}

func topologyChange(pk uint64) db.DeploymentChange {
	return db.DeploymentChange{
		Pk:           pk,
		ResourceType: db.DeploymentChangesResourceTypeDeploymentTopology,
		ResourceID:   "deploy_test",
		RegionID:     "region_test",
	}
}

// A not-found row is skipped with a bare version event so the cursor advances.
func TestFetchDeploymentChangeEvents_NotFoundSkipsAndAdvances(t *testing.T) {
	svc := &Service{db: &stubDatabase{
		changes: []db.DeploymentChange{topologyChange(42)},
		findErr: sql.ErrNoRows,
	}}

	events, err := svc.fetchDeploymentChangeEvents(context.Background(), "region_test", 0)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, uint64(42), events[0].GetVersion())
	require.Nil(t, events[0].GetEvent())
}

// A transient DB error must not advance the cursor: the fetch fails so the
// stream aborts and the client retries from its last seen version.
func TestFetchDeploymentChangeEvents_TransientErrorAborts(t *testing.T) {
	transient := errors.New("connection reset by peer")
	svc := &Service{db: &stubDatabase{
		changes: []db.DeploymentChange{topologyChange(42)},
		findErr: transient,
	}}

	events, err := svc.fetchDeploymentChangeEvents(context.Background(), "region_test", 0)
	require.Error(t, err)
	require.ErrorIs(t, err, transient)
	require.Nil(t, events)
}

// A row that can never be processed (unknown desired_status) is unrecoverable:
// it is skipped with a bare version event instead of blocking the stream.
func TestFetchDeploymentChangeEvents_UnrecoverableRowSkipsAndAdvances(t *testing.T) {
	svc := &Service{db: &stubDatabase{
		changes: []db.DeploymentChange{topologyChange(42)},
		findRow: db.FindDeploymentTopologyByDeploymentAndRegionRow{
			TopologyDesiredStatus: db.DeploymentTopologyDesiredStatus("bogus"),
		},
	}}

	events, err := svc.fetchDeploymentChangeEvents(context.Background(), "region_test", 0)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, uint64(42), events[0].GetVersion())
	require.Nil(t, events[0].GetEvent())
}

// A loadable running row produces a full deployment event.
func TestFetchDeploymentChangeEvents_Success(t *testing.T) {
	svc := &Service{db: &stubDatabase{
		changes: []db.DeploymentChange{topologyChange(42)},
		findRow: db.FindDeploymentTopologyByDeploymentAndRegionRow{
			TopologyDesiredStatus: db.DeploymentTopologyDesiredStatusRunning,
			DeploymentID:          "deploy_test",
			DeploymentK8sName:     "k8s-test",
		},
	}}

	events, err := svc.fetchDeploymentChangeEvents(context.Background(), "region_test", 0)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, uint64(42), events[0].GetVersion())
	require.NotNil(t, events[0].GetDeployment())
	require.NotNil(t, events[0].GetDeployment().GetApply())
	require.Equal(t, "deploy_test", events[0].GetDeployment().GetApply().GetDeploymentId())
}
