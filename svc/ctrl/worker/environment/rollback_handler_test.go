package environment_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestRollbackDeploymentSwitchesLiveRoutesBack(t *testing.T) {
	f := newFixture(t)
	f.scheduleStop(t, f.candidate.ID)

	require.NoError(t, f.rollback(f.env.ID, f.live.ID, f.candidate.ID))

	f.requireLive(t, f.candidate.ID, true)
	f.requireRoutes(t, f.candidate.ID, f.live.ID)
	f.requireDesiredStateAfterStopDelay(t, f.candidate.ID, mysqltype.DeploymentsDesiredStateRunning)
	require.Equal(t, 1, countAudits(t, f.ctx, f.db, f.workspaceID, auditlog.DeploymentRollbackEvent, f.candidate.ID, f.actorID))
	f.requireAuditTargets(t, auditlog.DeploymentRollbackEvent, map[string]string{
		f.candidate.ID: "deployment",
		f.app.ID:       "app",
		f.live.ID:      "deployment",
	})
}

func TestRollbackDeploymentRetiresSource(t *testing.T) {
	for _, tc := range []struct {
		name           string
		promoteAnother bool
	}{
		{name: "rollback"},
		{name: "rollback then promotion", promoteAnother: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			deploymentIDs := []string{f.live.ID, f.candidate.ID, f.other.ID}
			var regionIDs []string
			for _, name := range []string{"us-west-2", "us-east-1"} {
				region := f.seeder.CreateRegion(f.ctx, seed.CreateRegionRequest{Name: name, Platform: "aws"})
				regionIDs = append(regionIDs, region.ID)
				for _, deploymentID := range deploymentIDs {
					require.NoError(t, f.db.InsertDeploymentTopology(f.ctx, db.InsertDeploymentTopologyParams{
						WorkspaceID:            f.workspaceID,
						DeploymentID:           deploymentID,
						RegionID:               region.ID,
						AutoscalingReplicasMin: 1,
						AutoscalingReplicasMax: 1,
						DesiredStatus:          db.DeploymentTopologyDesiredStatusRunning,
						CreatedAt:              time.Now().UnixMilli(),
					}))
				}
			}

			require.NoError(t, f.rollback(f.env.ID, f.live.ID, f.candidate.ID))
			require.Never(t, func() bool {
				source, err := f.db.FindDeploymentById(f.ctx, f.live.ID)
				return err != nil || source.DesiredState != mysqltype.DeploymentsDesiredStateRunning
			}, 300*time.Millisecond, 25*time.Millisecond, "source stays running during standby")

			liveID := f.candidate.ID
			if tc.promoteAnother {
				require.NoError(t, f.promote(f.env.ID, f.other.ID))
				liveID = f.other.ID
			}

			require.EventuallyWithT(t, func(t *assert.CollectT) {
				for _, deploymentID := range deploymentIDs {
					wantState := mysqltype.DeploymentsDesiredStateRunning
					wantTopology := db.DeploymentTopologyDesiredStatusRunning
					if deploymentID == f.live.ID || (tc.promoteAnother && deploymentID == f.candidate.ID) {
						wantState = mysqltype.DeploymentsDesiredStateStopped
						wantTopology = db.DeploymentTopologyDesiredStatusStopped
					}
					dep, err := f.db.FindDeploymentById(f.ctx, deploymentID)
					require.NoError(t, err)
					require.Equal(t, wantState, dep.DesiredState, "deployment %s", deploymentID)
					for _, regionID := range regionIDs {
						topology, err := f.db.FindDeploymentTopologyByDeploymentAndRegion(f.ctx, db.FindDeploymentTopologyByDeploymentAndRegionParams{
							DeploymentID: deploymentID,
							RegionID:     regionID,
						})
						require.NoError(t, err)
						require.Equal(t, wantTopology, topology.DesiredStatus, "deployment %s region %s", deploymentID, regionID)
					}
				}
			}, 10*time.Second, 50*time.Millisecond)
			f.requireLive(t, liveID, !tc.promoteAnother)
			f.requireRoutes(t, liveID, f.live.ID)
		})
	}
}

func TestRollbackDeploymentRestoresSourceAndRetiresItOnNextPromotion(t *testing.T) {
	for _, action := range []string{"promote", "rollback"} {
		t.Run(action, func(t *testing.T) {
			f := newFixture(t)
			require.NoError(t, f.rollback(f.env.ID, f.live.ID, f.candidate.ID))
			if action == "promote" {
				require.NoError(t, f.promote(f.env.ID, f.live.ID))
			} else {
				require.NoError(t, f.rollback(f.env.ID, f.candidate.ID, f.live.ID))
			}

			require.EventuallyWithT(t, func(t *assert.CollectT) {
				demoted, err := f.db.FindDeploymentById(f.ctx, f.candidate.ID)
				require.NoError(t, err)
				require.Equal(t, mysqltype.DeploymentsDesiredStateStopped, demoted.DesiredState)
			}, 10*time.Second, 50*time.Millisecond)
			restored, err := f.db.FindDeploymentById(f.ctx, f.live.ID)
			require.NoError(t, err)
			require.Equal(t, mysqltype.DeploymentsDesiredStateRunning, restored.DesiredState)
			f.requireLive(t, f.live.ID, action == "rollback")
			f.requireRoutes(t, f.live.ID, f.live.ID)

			require.NoError(t, f.promote(f.env.ID, f.other.ID))
			require.Never(t, func() bool {
				source, err := f.db.FindDeploymentById(f.ctx, f.live.ID)
				return err != nil || source.DesiredState != mysqltype.DeploymentsDesiredStateRunning
			}, 300*time.Millisecond, 25*time.Millisecond, "restored source gets a new standby period")
			require.EventuallyWithT(t, func(t *assert.CollectT) {
				source, err := f.db.FindDeploymentById(f.ctx, f.live.ID)
				require.NoError(t, err)
				require.Equal(t, mysqltype.DeploymentsDesiredStateStopped, source.DesiredState)
			}, 10*time.Second, 50*time.Millisecond)
			f.requireLive(t, f.other.ID, false)
			f.requireRoutes(t, f.other.ID, f.live.ID)
		})
	}
}

func TestRollbackDeploymentRejectsStaleSource(t *testing.T) {
	f := newFixture(t)

	err := f.rollback(f.env.ID, f.other.ID, f.candidate.ID)
	require.ErrorContains(t, err, "no longer live")

	f.requireLive(t, f.live.ID, false)
	f.requireRoutes(t, f.live.ID, f.live.ID)
	require.Equal(t, 0, countAudits(t, f.ctx, f.db, f.workspaceID, auditlog.DeploymentRollbackEvent, f.candidate.ID, f.actorID))
}

func TestRollbackDeploymentRejectsForeignEnvironment(t *testing.T) {
	f := newFixture(t)
	foreign := f.newProductionEnvironment(t, "production-kebap")

	err := f.rollback(foreign.ID, f.live.ID, f.candidate.ID)
	require.ErrorContains(t, err, "keyed environment")

	f.requireLive(t, f.live.ID, false)
	f.requireRoutes(t, f.live.ID, f.live.ID)
	require.Equal(t, 0, countAudits(t, f.ctx, f.db, f.workspaceID, auditlog.DeploymentRollbackEvent, f.candidate.ID, f.actorID))
}
