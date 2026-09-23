package cluster

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	dbtype "github.com/unkeyed/unkey/pkg/mysql/types"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestReportInstanceEventsFiltersStartupFailures(t *testing.T) {
	for _, tt := range []struct {
		name          string
		status        dbtype.DeploymentsStatus
		reason        string
		cause         ctrlv1.TerminationCause
		waiting       *ctrlv1.Waiting
		container     string
		live          bool
		missingID     bool
		deliveryFails bool
		deliveryHangs bool
		wantNotifies  int32
	}{
		{name: "first OOM without diagnostic reason", status: dbtype.DeploymentsStatusDeploying, cause: ctrlv1.TerminationCause_TERMINATION_CAUSE_OOM_KILLED, container: "deployment", live: true, wantNotifies: 1},
		{name: "retry identical OOM after invocation ID is saved", status: dbtype.DeploymentsStatusDeploying, cause: ctrlv1.TerminationCause_TERMINATION_CAUSE_OOM_KILLED, container: "deployment", live: true, missingID: true, wantNotifies: 1},
		{name: "cancelled without invocation ID", status: dbtype.DeploymentsStatusCancelled, cause: ctrlv1.TerminationCause_TERMINATION_CAUSE_OOM_KILLED, container: "deployment", missingID: true},
		{name: "exit 137 without OOM", status: dbtype.DeploymentsStatusDeploying, reason: "Error", cause: ctrlv1.TerminationCause_TERMINATION_CAUSE_OTHER, container: "deployment", live: true},
		{name: "unclassified event from older Krane", status: dbtype.DeploymentsStatusDeploying, reason: "OOMKilled", container: "deployment", live: true},
		{name: "unknown cause", status: dbtype.DeploymentsStatusDeploying, reason: "OOMKilled", cause: ctrlv1.TerminationCause(99), container: "deployment", live: true},
		{name: "unclassified waiting state", status: dbtype.DeploymentsStatusDeploying, waiting: &ctrlv1.Waiting{Reason: "InvalidImageName"}, container: "deployment", live: true},
		{name: "unknown waiting cause", status: dbtype.DeploymentsStatusDeploying, waiting: &ctrlv1.Waiting{Reason: "InvalidImageName", Cause: ctrlv1.WaitingCause(99)}, container: "deployment", live: true},
		{name: "transient image pull backoff", status: dbtype.DeploymentsStatusDeploying, waiting: &ctrlv1.Waiting{Reason: "ImagePullBackOff"}, container: "deployment", live: true},
		{name: "container config error remains diagnostic", status: dbtype.DeploymentsStatusDeploying, waiting: &ctrlv1.Waiting{Reason: "CreateContainerConfigError", Cause: ctrlv1.WaitingCause_WAITING_CAUSE_CONTAINER_CONFIG_ERROR}, container: "deployment", live: true},
		{name: "other container", status: dbtype.DeploymentsStatusDeploying, reason: "OOMKilled", cause: ctrlv1.TerminationCause_TERMINATION_CAUSE_OOM_KILLED, container: "sidecar", live: true},
		{name: "already live", status: dbtype.DeploymentsStatusReady, reason: "OOMKilled", cause: ctrlv1.TerminationCause_TERMINATION_CAUSE_OOM_KILLED, container: "deployment", live: true},
		{name: "cancelled", status: dbtype.DeploymentsStatusCancelled, reason: "OOMKilled", cause: ctrlv1.TerminationCause_TERMINATION_CAUSE_OOM_KILLED, container: "deployment", live: true},
		{name: "routing already started", status: dbtype.DeploymentsStatusNetwork, reason: "OOMKilled", cause: ctrlv1.TerminationCause_TERMINATION_CAUSE_OOM_KILLED, container: "deployment", live: true},
		{name: "wake after completed deploy", status: dbtype.DeploymentsStatusDeploying, reason: "OOMKilled", cause: ctrlv1.TerminationCause_TERMINATION_CAUSE_OOM_KILLED, container: "deployment", live: false},
		{name: "retry failed delivery", status: dbtype.DeploymentsStatusDeploying, reason: "OOMKilled", cause: ctrlv1.TerminationCause_TERMINATION_CAUSE_OOM_KILLED, container: "deployment", live: true, deliveryFails: true, wantNotifies: 1},
		{name: "bound stalled acceptance", status: dbtype.DeploymentsStatusDeploying, cause: ctrlv1.TerminationCause_TERMINATION_CAUSE_OOM_KILLED, container: "deployment", live: true, deliveryHangs: true, wantNotifies: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			deploymentID := uid.New(uid.DeploymentPrefix)
			invocationID := uid.New("inv")
			var notifications atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/query" {
					rows := []map[string]string{}
					if tt.live {
						rows = append(rows, map[string]string{"id": invocationID})
					}
					require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"rows": rows}))
					return
				}
				if r.URL.Path != "/hydra.v1.DeployWorkflow/"+deploymentID+"/NotifyReadiness/send" {
					http.NotFound(w, r)
					return
				}
				var req struct {
					State string `json:"state"`
				}
				require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
				require.Equal(t, "STATE_OUT_OF_MEMORY", req.State)
				notifications.Add(1)
				if tt.deliveryFails {
					http.Error(w, "unavailable", http.StatusServiceUnavailable)
					return
				}
				if tt.deliveryHangs {
					<-r.Context().Done()
					return
				}
				w.WriteHeader(http.StatusAccepted)
				require.NoError(t, json.NewEncoder(w).Encode(map[string]string{
					"invocationId": uid.New("inv"),
					"status":       "Accepted",
				}))
			}))
			t.Cleanup(server.Close)
			database := &instanceEventsDatabase{deployment: db.Deployment{
				ID: deploymentID, Status: tt.status,
				InvocationID: sql.NullString{String: invocationID, Valid: !tt.missingID},
			}}
			svc := &Service{
				db:             database,
				bearer:         "test-token",
				restate:        ingress.NewClient(server.URL),
				restateAdmin:   restateadmin.New(restateadmin.Config{BaseURL: server.URL}),
				clusterCache:   cache.NewNoopCache[clusterCacheKey, db.FindClusterRow](),
				instanceEvents: batch.NewNoop[schema.InstanceEventV1](),
			}
			req := connect.NewRequest(&ctrlv1.ReportInstanceEventsRequest{
				Cluster: &ctrlv1.ClusterKey{CellId: "cell", Platform: "aws", Region: "eu-west-1"},
				Events: []*ctrlv1.InstanceEvent{nil, {
					DeploymentId: deploymentID, ContainerName: tt.container, RestartCount: 0,
					State: &ctrlv1.InstanceEvent_Terminated{Terminated: &ctrlv1.Terminated{
						Reason: tt.reason, ExitCode: 137, Signal: 9, Cause: tt.cause,
					}},
				}},
			})
			if tt.waiting != nil {
				req.Msg.Events[1].State = &ctrlv1.InstanceEvent_Waiting{Waiting: tt.waiting}
			}
			req.Header().Set("Authorization", "Bearer test-token")
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			t.Cleanup(cancel)
			_, err := svc.ReportInstanceEvents(ctx, req)
			if tt.missingID && tt.status == dbtype.DeploymentsStatusDeploying {
				require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
				require.Zero(t, notifications.Load())
				database.deployment.InvocationID.Valid = true
				_, err = svc.ReportInstanceEvents(ctx, req)
			}
			require.NoError(t, ctx.Err(), "notification must not consume the caller's entire timeout")
			if tt.deliveryFails || tt.deliveryHangs {
				require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.wantNotifies, notifications.Load())
		})
	}
}

type instanceEventsDatabase struct {
	db.Database
	deployment db.Deployment
}

func (d *instanceEventsDatabase) FindCluster(context.Context, db.FindClusterParams) (db.FindClusterRow, error) {
	return db.FindClusterRow{RegionID: "region", RegionName: "eu-west-1", RegionPlatform: "aws"}, nil
}

func (d *instanceEventsDatabase) FindDeploymentById(context.Context, string) (db.Deployment, error) {
	return d.deployment, nil
}

func (d *instanceEventsDatabase) RecordInstanceExit(context.Context, db.RecordInstanceExitParams) error {
	return nil
}
