package cluster

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
)

// fullDeploymentRow returns a deploymentRow with every field that
// deploymentRowToState reads populated with a non-zero value, so the producer
// coverage guard can verify each ApplyDeployment proto field is actually filled
// in from the database row.
func fullDeploymentRow() deploymentRow {
	return deploymentRow{
		dt: db.DeploymentTopology{
			DesiredStatus:              db.DeploymentTopologyDesiredStatusRunning,
			AutoscalingReplicasMin:     2,
			AutoscalingReplicasMax:     5,
			AutoscalingThresholdCpu:    sql.NullInt16{Valid: true, Int16: 80},
			AutoscalingThresholdMemory: sql.NullInt16{Valid: true, Int16: 75},
		},
		d: db.Deployment{
			ID:                            "deploy_sentinel",
			K8sName:                       "k8s-name-sentinel",
			WorkspaceID:                   "ws_sentinel",
			ProjectID:                     "prj_sentinel",
			EnvironmentID:                 "env_sentinel",
			AppID:                         "app_sentinel",
			Image:                         sql.NullString{Valid: true, String: "registry.io/sentinel:v1"},
			CpuMillicores:                 250,
			MemoryMib:                     256,
			StorageMib:                    2048,
			EncryptedEnvironmentVariables: []byte("ciphertext-sentinel"),
			BuildID:                       sql.NullString{Valid: true, String: "build_sentinel"},
			Command:                       dbtype.StringSlice{"/sentinel-app", "serve"},
			Port:                          8080,
			ShutdownSignal:                db.DeploymentsShutdownSignalSIGTERM,
			GitCommitSha:                  sql.NullString{Valid: true, String: "abc123sha"},
			GitBranch:                     sql.NullString{Valid: true, String: "main-sentinel"},
			GitCommitMessage:              sql.NullString{Valid: true, String: "sentinel commit"},
			Healthcheck: dbtype.NullHealthcheck{
				Valid:       true,
				Healthcheck: &dbtype.Healthcheck{Method: "GET", Path: "/sentinel-healthz"},
			},
		},
		k8sNamespace:    sql.NullString{Valid: true, String: "ns-sentinel"},
		environmentSlug: "production",
		regionName:      "us-east-1",
		gitRepo:         sql.NullString{Valid: true, String: "github.com/test/sentinel"},
	}
}

// producerFieldAssertions maps each ApplyDeployment proto field (by proto name)
// to an assertion that deploymentRowToState populated it from the database row.
// Every proto field must have an entry; the coverage test fails otherwise, so a
// field cannot be added or dropped on the producer side without a test. This
// mirrors the krane render-side guard in apply_test.go.
var producerFieldAssertions = map[string]func(t *testing.T, a *ctrlv1.ApplyDeployment){
	"k8s_namespace": func(t *testing.T, a *ctrlv1.ApplyDeployment) {
		require.Equal(t, "ns-sentinel", a.GetK8SNamespace())
	},
	"k8s_name": func(t *testing.T, a *ctrlv1.ApplyDeployment) {
		require.Equal(t, "k8s-name-sentinel", a.GetK8SName())
	},
	"workspace_id": func(t *testing.T, a *ctrlv1.ApplyDeployment) {
		require.Equal(t, "ws_sentinel", a.GetWorkspaceId())
	},
	"project_id": func(t *testing.T, a *ctrlv1.ApplyDeployment) {
		require.Equal(t, "prj_sentinel", a.GetProjectId())
	},
	"environment_id": func(t *testing.T, a *ctrlv1.ApplyDeployment) {
		require.Equal(t, "env_sentinel", a.GetEnvironmentId())
	},
	"deployment_id": func(t *testing.T, a *ctrlv1.ApplyDeployment) {
		require.Equal(t, "deploy_sentinel", a.GetDeploymentId())
	},
	"image": func(t *testing.T, a *ctrlv1.ApplyDeployment) {
		require.Equal(t, "registry.io/sentinel:v1", a.GetImage())
	},
	"cpu_millicores": func(t *testing.T, a *ctrlv1.ApplyDeployment) {
		require.Equal(t, int64(250), a.GetCpuMillicores())
	},
	"memory_mib": func(t *testing.T, a *ctrlv1.ApplyDeployment) {
		require.Equal(t, int64(256), a.GetMemoryMib())
	},
	"build_id": func(t *testing.T, a *ctrlv1.ApplyDeployment) {
		require.Equal(t, "build_sentinel", a.GetBuildId())
	},
	"encrypted_environment_variables": func(t *testing.T, a *ctrlv1.ApplyDeployment) {
		require.NotEmpty(t, a.GetEncryptedEnvironmentVariables())
	},
	"command": func(t *testing.T, a *ctrlv1.ApplyDeployment) {
		require.Equal(t, []string{"/sentinel-app", "serve"}, a.GetCommand(),
			"command must be carried from the DB row")
	},
	"port": func(t *testing.T, a *ctrlv1.ApplyDeployment) {
		require.Equal(t, int32(8080), a.GetPort())
	},
	"shutdown_signal": func(t *testing.T, a *ctrlv1.ApplyDeployment) {
		require.NotEmpty(t, a.GetShutdownSignal())
	},
	"healthcheck": func(t *testing.T, a *ctrlv1.ApplyDeployment) {
		require.NotEmpty(t, a.GetHealthcheck())
	},
	"app_id": func(t *testing.T, a *ctrlv1.ApplyDeployment) {
		require.Equal(t, "app_sentinel", a.GetAppId())
	},
	"environment_slug": func(t *testing.T, a *ctrlv1.ApplyDeployment) {
		require.Equal(t, "production", a.GetEnvironmentSlug())
	},
	"region": func(t *testing.T, a *ctrlv1.ApplyDeployment) {
		require.Equal(t, "us-east-1", a.GetRegion())
	},
	"git_commit_sha": func(t *testing.T, a *ctrlv1.ApplyDeployment) {
		require.Equal(t, "abc123sha", a.GetGitCommitSha())
	},
	"git_branch": func(t *testing.T, a *ctrlv1.ApplyDeployment) {
		require.Equal(t, "main-sentinel", a.GetGitBranch())
	},
	"git_repo": func(t *testing.T, a *ctrlv1.ApplyDeployment) {
		require.Equal(t, "github.com/test/sentinel", a.GetGitRepo())
	},
	"git_commit_message": func(t *testing.T, a *ctrlv1.ApplyDeployment) {
		require.Equal(t, "sentinel commit", a.GetGitCommitMessage())
	},
	"autoscaling": func(t *testing.T, a *ctrlv1.ApplyDeployment) {
		require.NotNil(t, a.GetAutoscaling())
		require.Equal(t, uint32(2), a.GetAutoscaling().GetMinReplicas())
		require.Equal(t, uint32(5), a.GetAutoscaling().GetMaxReplicas())
	},
	"ephemeral_storage": func(t *testing.T, a *ctrlv1.ApplyDeployment) {
		require.NotNil(t, a.GetEphemeralStorage())
		require.Equal(t, int64(2048), a.GetEphemeralStorage().GetSizeMib())
	},
}

// TestDeploymentRowToState_PopulatesProtoFields converts a fully-populated row
// and asserts each ApplyDeployment proto field was carried over from the DB.
func TestDeploymentRowToState_PopulatesProtoFields(t *testing.T) {
	state, err := deploymentRowToState(fullDeploymentRow(), 1)
	require.NoError(t, err)

	apply := state.GetApply()
	require.NotNil(t, apply)

	for field, assert := range producerFieldAssertions {
		t.Run(field, func(t *testing.T) {
			assert(t, apply)
		})
	}
}

// TestApplyDeploymentProducerFieldCoverage enumerates every field in the
// ApplyDeployment proto and fails if any is not covered by
// producerFieldAssertions, so a field cannot be dropped from the DB-to-proto
// mapping in deploymentRowToState without a failing test naming it.
func TestApplyDeploymentProducerFieldCoverage(t *testing.T) {
	fields := (&ctrlv1.ApplyDeployment{}).ProtoReflect().Descriptor().Fields()

	for i := 0; i < fields.Len(); i++ {
		name := string(fields.Get(i).Name())
		_, ok := producerFieldAssertions[name]
		require.Truef(t, ok,
			"ApplyDeployment proto field %q is not covered by a producer test. Map it "+
				"from the deployment row in deploymentRowToState and add an entry to "+
				"producerFieldAssertions.", name)
	}
}
