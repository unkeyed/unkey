package deployment

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/ptr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// Sentinel values for every ApplyDeployment field. Each is distinctive so the
// assertions can prove the value reached the rendered ReplicaSet.
const (
	testNamespace        = "test-ns"
	testK8sName          = "test-k8s-name"
	testWorkspaceID      = "ws_sentinel"
	testProjectID        = "proj_sentinel"
	testEnvironmentID    = "env_sentinel"
	testDeploymentID     = "dep_sentinel"
	testImage            = "registry.test/sentinel-image:v1"
	testCPUMillicores    = int64(1000)
	testMemoryMib        = int64(512)
	testBuildID          = "build_sentinel"
	testPort             = int32(8080)
	testShutdownSignal   = "SIGINT"
	testAppID            = "app_sentinel"
	testEnvironmentSlug  = "production"
	testRegion           = "us-east-1"
	testGitCommitSha     = "abc123sha"
	testGitBranch        = "main-sentinel"
	testGitRepo          = "github.com/test/sentinel"
	testGitCommitMessage = "sentinel commit message"
	testHealthcheckPath  = "/sentinel-healthz"
	testEphemeralMib     = int64(2048)
)

var testCommand = []string{"/sentinel-app", "serve", "--flag"}

// fullApplyRequest returns an ApplyDeployment with every field populated with a
// sentinel value. The field-coverage guard depends on this populating every
// field, so when the proto gains a field this helper must be updated too.
func fullApplyRequest(t *testing.T) *ctrlv1.ApplyDeployment {
	t.Helper()

	hc, err := json.Marshal(dbtype.Healthcheck{
		Method:              "GET",
		Path:                testHealthcheckPath,
		IntervalSeconds:     10,
		TimeoutSeconds:      2,
		FailureThreshold:    3,
		InitialDelaySeconds: 5,
	})
	require.NoError(t, err)

	return &ctrlv1.ApplyDeployment{
		K8SNamespace:                  testNamespace,
		K8SName:                       testK8sName,
		WorkspaceId:                   testWorkspaceID,
		ProjectId:                     testProjectID,
		EnvironmentId:                 testEnvironmentID,
		DeploymentId:                  testDeploymentID,
		Image:                         testImage,
		CpuMillicores:                 testCPUMillicores,
		MemoryMib:                     testMemoryMib,
		BuildId:                       ptr.P(testBuildID),
		EncryptedEnvironmentVariables: []byte("ciphertext-sentinel"),
		Command:                       testCommand,
		Port:                          testPort,
		ShutdownSignal:                testShutdownSignal,
		Healthcheck:                   hc,
		AppId:                         testAppID,
		EnvironmentSlug:               ptr.P(testEnvironmentSlug),
		Region:                        ptr.P(testRegion),
		GitCommitSha:                  ptr.P(testGitCommitSha),
		GitBranch:                     ptr.P(testGitBranch),
		GitRepo:                       ptr.P(testGitRepo),
		GitCommitMessage:              ptr.P(testGitCommitMessage),
		Autoscaling:                   &ctrlv1.AutoscalingPolicy{MinReplicas: 2, MaxReplicas: 5},
		EphemeralStorage:              &ctrlv1.EphemeralStorage{SizeMib: testEphemeralMib},
	}
}

func testController() *Controller {
	return &Controller{
		platform:         "test-platform",
		storageClassName: "test-storage-class",
		imagePullSecrets: []corev1.LocalObjectReference{{Name: "pull-secret"}},
	}
}

func mainContainer(t *testing.T, rs *appsv1.ReplicaSet) corev1.Container {
	t.Helper()
	require.Len(t, rs.Spec.Template.Spec.Containers, 1, "expected exactly one container")
	return rs.Spec.Template.Spec.Containers[0]
}

func envValue(c corev1.Container, name string) (string, bool) {
	for _, e := range c.Env {
		if e.Name == name {
			return e.Value, true
		}
	}
	return "", false
}

func hasLabelValue(labels map[string]string, want string) bool {
	for _, v := range labels {
		if v == want {
			return true
		}
	}
	return false
}

// fieldAssertions maps each ApplyDeployment proto field (by proto name) to an
// assertion that it is wired into the rendered ReplicaSet. Together with
// fieldsRenderedElsewhere it must cover every proto field; the coverage test
// fails otherwise, so a field cannot be added or dropped without a test.
var fieldAssertions = map[string]func(t *testing.T, rs *appsv1.ReplicaSet){
	"k8s_namespace": func(t *testing.T, rs *appsv1.ReplicaSet) {
		require.Equal(t, testNamespace, rs.Namespace)
	},
	"k8s_name": func(t *testing.T, rs *appsv1.ReplicaSet) {
		require.Equal(t, testK8sName, rs.Name)
		require.Equal(t, testK8sName+"-", rs.Spec.Template.GenerateName)
	},
	"workspace_id": func(t *testing.T, rs *appsv1.ReplicaSet) {
		require.True(t, hasLabelValue(rs.Labels, testWorkspaceID), "workspace_id must appear as a label")
	},
	"project_id": func(t *testing.T, rs *appsv1.ReplicaSet) {
		require.True(t, hasLabelValue(rs.Labels, testProjectID), "project_id must appear as a label")
	},
	"environment_id": func(t *testing.T, rs *appsv1.ReplicaSet) {
		require.True(t, hasLabelValue(rs.Labels, testEnvironmentID), "environment_id must appear as a label")
	},
	"deployment_id": func(t *testing.T, rs *appsv1.ReplicaSet) {
		require.True(t, hasLabelValue(rs.Labels, testDeploymentID), "deployment_id must appear as a label")
		require.Equal(t, testDeploymentID, rs.Spec.Selector.MatchLabels[labelDeploymentIDKey(t)])
		v, ok := envValue(mainContainer(t, rs), "UNKEY_DEPLOYMENT_ID")
		require.True(t, ok)
		require.Equal(t, testDeploymentID, v)
	},
	"image": func(t *testing.T, rs *appsv1.ReplicaSet) {
		require.Equal(t, testImage, mainContainer(t, rs).Image)
	},
	"cpu_millicores": func(t *testing.T, rs *appsv1.ReplicaSet) {
		cpu := mainContainer(t, rs).Resources.Limits[corev1.ResourceCPU]
		require.Equal(t, "1", cpu.String(), "1000m CPU limit normalizes to 1")
	},
	"memory_mib": func(t *testing.T, rs *appsv1.ReplicaSet) {
		mem := mainContainer(t, rs).Resources.Limits[corev1.ResourceMemory]
		require.Equal(t, "512Mi", mem.String())
	},
	"build_id": func(t *testing.T, rs *appsv1.ReplicaSet) {
		require.True(t, hasLabelValue(rs.Labels, testBuildID), "build_id must appear as a label")
	},
	"encrypted_environment_variables": func(t *testing.T, rs *appsv1.ReplicaSet) {
		// Decrypted into a K8s Secret outside buildReplicaSet; its effect here is
		// the envFrom secretRef mount, gated on hasSecrets.
		c := mainContainer(t, rs)
		require.Len(t, c.EnvFrom, 1, "secret env vars must be mounted via envFrom")
		require.NotNil(t, c.EnvFrom[0].SecretRef)
	},
	"command": func(t *testing.T, rs *appsv1.ReplicaSet) {
		require.Equal(t, testCommand, mainContainer(t, rs).Command,
			"command override must be applied to the container")
	},
	"port": func(t *testing.T, rs *appsv1.ReplicaSet) {
		c := mainContainer(t, rs)
		require.Len(t, c.Ports, 1)
		require.Equal(t, testPort, c.Ports[0].ContainerPort)
		v, ok := envValue(c, "PORT")
		require.True(t, ok)
		require.Equal(t, "8080", v)
	},
	"shutdown_signal": func(t *testing.T, rs *appsv1.ReplicaSet) {
		c := mainContainer(t, rs)
		require.NotNil(t, c.Lifecycle)
		require.NotNil(t, c.Lifecycle.PreStop)
		require.NotNil(t, c.Lifecycle.PreStop.Exec)
		require.Contains(t, c.Lifecycle.PreStop.Exec.Command, "-SIGINT")
	},
	"healthcheck": func(t *testing.T, rs *appsv1.ReplicaSet) {
		c := mainContainer(t, rs)
		require.NotNil(t, c.LivenessProbe)
		require.NotNil(t, c.ReadinessProbe)
		require.NotNil(t, c.LivenessProbe.HTTPGet)
		require.Equal(t, testHealthcheckPath, c.LivenessProbe.HTTPGet.Path)
	},
	"app_id": func(t *testing.T, rs *appsv1.ReplicaSet) {
		require.True(t, hasLabelValue(rs.Labels, testAppID), "app_id must appear as a label")
	},
	"environment_slug": func(t *testing.T, rs *appsv1.ReplicaSet) {
		v, ok := envValue(mainContainer(t, rs), "UNKEY_ENVIRONMENT_SLUG")
		require.True(t, ok)
		require.Equal(t, testEnvironmentSlug, v)
	},
	"region": func(t *testing.T, rs *appsv1.ReplicaSet) {
		v, ok := envValue(mainContainer(t, rs), "UNKEY_REGION")
		require.True(t, ok)
		require.Equal(t, testRegion, v)
	},
	"git_commit_sha": func(t *testing.T, rs *appsv1.ReplicaSet) {
		v, ok := envValue(mainContainer(t, rs), "UNKEY_GIT_COMMIT_SHA")
		require.True(t, ok)
		require.Equal(t, testGitCommitSha, v)
	},
	"git_branch": func(t *testing.T, rs *appsv1.ReplicaSet) {
		v, ok := envValue(mainContainer(t, rs), "UNKEY_GIT_BRANCH")
		require.True(t, ok)
		require.Equal(t, testGitBranch, v)
	},
	"git_repo": func(t *testing.T, rs *appsv1.ReplicaSet) {
		v, ok := envValue(mainContainer(t, rs), "UNKEY_GIT_REPO")
		require.True(t, ok)
		require.Equal(t, testGitRepo, v)
	},
	"git_commit_message": func(t *testing.T, rs *appsv1.ReplicaSet) {
		v, ok := envValue(mainContainer(t, rs), "UNKEY_GIT_COMMIT_MESSAGE")
		require.True(t, ok)
		require.Equal(t, testGitCommitMessage, v)
	},
	"ephemeral_storage": func(t *testing.T, rs *appsv1.ReplicaSet) {
		var found bool
		for _, vol := range rs.Spec.Template.Spec.Volumes {
			if vol.Name == "data" && vol.Ephemeral != nil {
				found = true
			}
		}
		require.True(t, found, "ephemeral_storage must produce a generic ephemeral volume")
		var mounted bool
		for _, m := range mainContainer(t, rs).VolumeMounts {
			if m.MountPath == "/data" {
				mounted = true
			}
		}
		require.True(t, mounted, "ephemeral volume must be mounted at /data")
	},
}

// fieldsRenderedElsewhere lists proto fields that intentionally do not surface
// in the ReplicaSet, with the reason.
var fieldsRenderedElsewhere = map[string]string{
	"autoscaling": "rendered into a HorizontalPodAutoscaler by ensureHPAExists, not the ReplicaSet",
}

// labelDeploymentIDKey returns the label key used for the deployment id by
// rendering a known value and reading it back, so the test does not hardcode
// the label package's internal key string.
func labelDeploymentIDKey(t *testing.T) string {
	t.Helper()
	rs := testController().buildReplicaSet(fullApplyRequest(t), true)
	for k, v := range rs.Spec.Selector.MatchLabels {
		if v == testDeploymentID {
			return k
		}
	}
	t.Fatal("deployment id label key not found in selector")
	return ""
}

// TestBuildReplicaSet_WiresProtoFields renders a fully-populated request and
// asserts each proto field surfaces in the ReplicaSet.
func TestBuildReplicaSet_WiresProtoFields(t *testing.T) {
	rs := testController().buildReplicaSet(fullApplyRequest(t), true /* hasSecrets */)

	for field, assert := range fieldAssertions {
		t.Run(field, func(t *testing.T) {
			assert(t, rs)
		})
	}
}

// TestApplyDeploymentFieldCoverage enumerates every field in the
// ApplyDeployment proto and fails if any is not covered by fieldAssertions or
// fieldsRenderedElsewhere, so a field cannot be dropped from the render path
// without a failing test naming it.
func TestApplyDeploymentFieldCoverage(t *testing.T) {
	fields := (&ctrlv1.ApplyDeployment{}).ProtoReflect().Descriptor().Fields()

	for i := 0; i < fields.Len(); i++ {
		name := string(fields.Get(i).Name())

		_, asserted := fieldAssertions[name]
		_, elsewhere := fieldsRenderedElsewhere[name]

		require.Truef(t, asserted || elsewhere,
			"ApplyDeployment proto field %q is not covered by a test. Wire it into "+
				"buildReplicaSet and add an entry to fieldAssertions, or document it in "+
				"fieldsRenderedElsewhere.", name)

		require.Falsef(t, asserted && elsewhere,
			"ApplyDeployment proto field %q is in both fieldAssertions and "+
				"fieldsRenderedElsewhere; it must be in exactly one.", name)
	}
}

// TestBuildReplicaSet_NoCommandUsesImageEntrypoint verifies the safe default:
// when no command override is provided, the container command stays nil so the
// image's ENTRYPOINT/CMD runs.
func TestBuildReplicaSet_NoCommandUsesImageEntrypoint(t *testing.T) {
	req := fullApplyRequest(t)
	req.Command = nil

	rs := testController().buildReplicaSet(req, true)
	require.Nil(t, mainContainer(t, rs).Command)
}

// TestBuildReplicaSet_NoSecretsOmitsEnvFrom verifies that without secrets the
// container has no envFrom mount and the pod uses no dedicated service account.
func TestBuildReplicaSet_NoSecretsOmitsEnvFrom(t *testing.T) {
	rs := testController().buildReplicaSet(fullApplyRequest(t), false)
	require.Empty(t, mainContainer(t, rs).EnvFrom)
	require.Empty(t, rs.Spec.Template.Spec.ServiceAccountName)
}
