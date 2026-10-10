package deployment

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// Sentinel values for every ApplyDeployment field. Each is distinctive so the
// assertions can prove the value reached the rendered Deployment.
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
		BuildId:                       new(testBuildID),
		EncryptedEnvironmentVariables: []byte("ciphertext-sentinel"),
		Command:                       testCommand,
		Port:                          testPort,
		ShutdownSignal:                testShutdownSignal,
		Healthcheck:                   hc,
		AppId:                         testAppID,
		EnvironmentSlug:               new(testEnvironmentSlug),
		Region:                        new(testRegion),
		GitCommitSha:                  new(testGitCommitSha),
		GitBranch:                     new(testGitBranch),
		GitRepo:                       new(testGitRepo),
		GitCommitMessage:              new(testGitCommitMessage),
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

func mainContainer(t *testing.T, dep *appsv1.Deployment) corev1.Container {
	t.Helper()
	require.Len(t, dep.Spec.Template.Spec.Containers, 1, "expected exactly one container")
	return dep.Spec.Template.Spec.Containers[0]
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

var fieldAssertions = map[string]func(t *testing.T, dep *appsv1.Deployment){
	"k8s_namespace": func(t *testing.T, dep *appsv1.Deployment) {
		require.Equal(t, testNamespace, dep.Namespace)
	},
	"k8s_name": func(t *testing.T, dep *appsv1.Deployment) {
		require.Equal(t, testK8sName, dep.Name)
		require.Equal(t, testK8sName+"-", dep.Spec.Template.GenerateName)
	},
	"workspace_id": func(t *testing.T, dep *appsv1.Deployment) {
		require.True(t, hasLabelValue(dep.Labels, testWorkspaceID), "workspace_id must appear as a label")
	},
	"project_id": func(t *testing.T, dep *appsv1.Deployment) {
		require.True(t, hasLabelValue(dep.Labels, testProjectID), "project_id must appear as a label")
	},
	"environment_id": func(t *testing.T, dep *appsv1.Deployment) {
		require.True(t, hasLabelValue(dep.Labels, testEnvironmentID), "environment_id must appear as a label")
	},
	"deployment_id": func(t *testing.T, dep *appsv1.Deployment) {
		require.True(t, hasLabelValue(dep.Labels, testDeploymentID), "deployment_id must appear as a label")
		require.Equal(t, testDeploymentID, dep.Spec.Selector.MatchLabels[labelDeploymentIDKey(t)])
		v, ok := envValue(mainContainer(t, dep), "UNKEY_DEPLOYMENT_ID")
		require.True(t, ok)
		require.Equal(t, testDeploymentID, v)
	},
	"image": func(t *testing.T, dep *appsv1.Deployment) {
		require.Equal(t, testImage, mainContainer(t, dep).Image)
	},
	"cpu_millicores": func(t *testing.T, dep *appsv1.Deployment) {
		cpu := mainContainer(t, dep).Resources.Limits[corev1.ResourceCPU]
		require.Equal(t, "1", cpu.String(), "1000m CPU limit normalizes to 1")
	},
	"memory_mib": func(t *testing.T, dep *appsv1.Deployment) {
		mem := mainContainer(t, dep).Resources.Limits[corev1.ResourceMemory]
		require.Equal(t, "512Mi", mem.String())
	},
	"build_id": func(t *testing.T, dep *appsv1.Deployment) {
		require.True(t, hasLabelValue(dep.Labels, testBuildID), "build_id must appear as a label")
	},
	"encrypted_environment_variables": func(t *testing.T, dep *appsv1.Deployment) {
		// Decrypted into a K8s Secret outside buildDeployment; its effect here is
		// the envFrom secretRef mount, gated on hasSecrets.
		c := mainContainer(t, dep)
		require.Len(t, c.EnvFrom, 1, "secret env vars must be mounted via envFrom")
		require.NotNil(t, c.EnvFrom[0].SecretRef)
	},
	"command": func(t *testing.T, dep *appsv1.Deployment) {
		require.Equal(t, testCommand, mainContainer(t, dep).Command,
			"command override must be applied to the container")
	},
	"port": func(t *testing.T, dep *appsv1.Deployment) {
		c := mainContainer(t, dep)
		require.Len(t, c.Ports, 1)
		require.Equal(t, testPort, c.Ports[0].ContainerPort)
		v, ok := envValue(c, "PORT")
		require.True(t, ok)
		require.Equal(t, "8080", v)
	},
	"shutdown_signal": func(t *testing.T, dep *appsv1.Deployment) {
		c := mainContainer(t, dep)
		require.NotNil(t, c.Lifecycle)
		require.NotNil(t, c.Lifecycle.PreStop)
		require.NotNil(t, c.Lifecycle.PreStop.Exec)
		require.Contains(t, c.Lifecycle.PreStop.Exec.Command[2], "kill -s INT 1")
	},
	"healthcheck": func(t *testing.T, dep *appsv1.Deployment) {
		c := mainContainer(t, dep)
		require.NotNil(t, c.LivenessProbe)
		require.NotNil(t, c.ReadinessProbe)
		require.NotNil(t, c.LivenessProbe.HTTPGet)
		require.Equal(t, testHealthcheckPath, c.LivenessProbe.HTTPGet.Path)
	},
	"app_id": func(t *testing.T, dep *appsv1.Deployment) {
		require.True(t, hasLabelValue(dep.Labels, testAppID), "app_id must appear as a label")
	},
	"environment_slug": func(t *testing.T, dep *appsv1.Deployment) {
		v, ok := envValue(mainContainer(t, dep), "UNKEY_ENVIRONMENT_SLUG")
		require.True(t, ok)
		require.Equal(t, testEnvironmentSlug, v)
	},
	"region": func(t *testing.T, dep *appsv1.Deployment) {
		v, ok := envValue(mainContainer(t, dep), "UNKEY_REGION")
		require.True(t, ok)
		require.Equal(t, testRegion, v)
	},
	"git_commit_sha": func(t *testing.T, dep *appsv1.Deployment) {
		v, ok := envValue(mainContainer(t, dep), "UNKEY_GIT_COMMIT_SHA")
		require.True(t, ok)
		require.Equal(t, testGitCommitSha, v)
	},
	"git_branch": func(t *testing.T, dep *appsv1.Deployment) {
		v, ok := envValue(mainContainer(t, dep), "UNKEY_GIT_BRANCH")
		require.True(t, ok)
		require.Equal(t, testGitBranch, v)
	},
	"git_repo": func(t *testing.T, dep *appsv1.Deployment) {
		v, ok := envValue(mainContainer(t, dep), "UNKEY_GIT_REPO")
		require.True(t, ok)
		require.Equal(t, testGitRepo, v)
	},
	"git_commit_message": func(t *testing.T, dep *appsv1.Deployment) {
		v, ok := envValue(mainContainer(t, dep), "UNKEY_GIT_COMMIT_MESSAGE")
		require.True(t, ok)
		require.Equal(t, testGitCommitMessage, v)
	},
	"autoscaling": func(t *testing.T, dep *appsv1.Deployment) {
		constraints := dep.Spec.Template.Spec.TopologySpreadConstraints
		require.Len(t, constraints, 3)
		require.Equal(t, corev1.DoNotSchedule, constraints[2].WhenUnsatisfiable)
		require.Equal(t, int32(2), constraints[2].MaxSkew)
		require.Equal(t, new(int32(3)), constraints[2].MinDomains)
	},
	"ephemeral_storage": func(t *testing.T, dep *appsv1.Deployment) {
		var found bool
		for _, vol := range dep.Spec.Template.Spec.Volumes {
			if vol.Name == "data" && vol.Ephemeral != nil {
				found = true
			}
		}
		require.True(t, found, "ephemeral_storage must produce a generic ephemeral volume")
		var mounted bool
		for _, m := range mainContainer(t, dep).VolumeMounts {
			if m.MountPath == "/data" {
				mounted = true
			}
		}
		require.True(t, mounted, "ephemeral volume must be mounted at /data")
	},
}

// labelDeploymentIDKey returns the label key used for the deployment id by
// rendering a known value and reading it back, so the test does not hardcode
// the label package's internal key string.
func labelDeploymentIDKey(t *testing.T) string {
	t.Helper()
	dep := testController().buildDeployment(fullApplyRequest(t), true)
	for k, v := range dep.Spec.Selector.MatchLabels {
		if v == testDeploymentID {
			return k
		}
	}
	t.Fatal("deployment id label key not found in selector")
	return ""
}

func TestBuildDeployment_WiresProtoFields(t *testing.T) {
	dep := testController().buildDeployment(fullApplyRequest(t), true)

	for field, assert := range fieldAssertions {
		t.Run(field, func(t *testing.T) {
			assert(t, dep)
		})
	}
}

func TestApplyDeploymentFieldCoverage(t *testing.T) {
	fields := (&ctrlv1.ApplyDeployment{}).ProtoReflect().Descriptor().Fields()

	for i := 0; i < fields.Len(); i++ {
		name := string(fields.Get(i).Name())
		require.Contains(t, fieldAssertions, name, "ApplyDeployment proto field must be covered by fieldAssertions")
	}
}

func TestBuildDeployment_TopologySpread(t *testing.T) {
	for _, tt := range []struct {
		name        string
		minReplicas uint32
		maxReplicas uint32
		// hardMaxSkew is the maxSkew of the DoNotSchedule hostname constraint,
		// or 0 when the deployment has no hard hostname constraint.
		hardMaxSkew int32
	}{
		{"single", 1, 1, 0},
		{"single_minimum_with_autoscaling", 1, 2, 1},
		{"three", 3, 3, 1},
		{"four", 2, 4, 2},
		{"five", 2, 5, 2},
		{"six", 2, 6, 2},
		{"seven", 2, 7, 3},
		{"sixteen", 2, 16, 6},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := fullApplyRequest(t)
			req.Autoscaling = &ctrlv1.AutoscalingPolicy{MinReplicas: tt.minReplicas, MaxReplicas: tt.maxReplicas}
			dep := testController().buildDeployment(req, false)
			require.Nil(t, dep.Spec.Replicas)
			constraints := dep.Spec.Template.Spec.TopologySpreadConstraints

			deploymentLabels := map[string]string{"unkey.com/deployment.id": testDeploymentID}
			if tt.hardMaxSkew > 0 {
				require.Len(t, constraints, 3)
				hard := constraints[2]
				require.Equal(t, "kubernetes.io/hostname", hard.TopologyKey)
				require.Equal(t, tt.hardMaxSkew, hard.MaxSkew)
				require.Equal(t, corev1.DoNotSchedule, hard.WhenUnsatisfiable)
				require.Equal(t, new(int32(3)), hard.MinDomains)
				require.Equal(t, new(corev1.NodeInclusionPolicyHonor), hard.NodeTaintsPolicy)
				require.NotNil(t, hard.LabelSelector)
				require.Equal(t, deploymentLabels, hard.LabelSelector.MatchLabels)
			} else {
				require.Len(t, constraints, 2)
			}

			hostname := constraints[0]
			require.Equal(t, "kubernetes.io/hostname", hostname.TopologyKey)
			require.Equal(t, int32(1), hostname.MaxSkew)
			require.Equal(t, corev1.ScheduleAnyway, hostname.WhenUnsatisfiable)
			require.Nil(t, hostname.MinDomains)
			require.Nil(t, hostname.NodeTaintsPolicy)
			require.NotNil(t, hostname.LabelSelector)
			require.Equal(t, deploymentLabels, hostname.LabelSelector.MatchLabels)

			zone := constraints[1]
			require.Equal(t, "topology.kubernetes.io/zone", zone.TopologyKey)
			require.Equal(t, int32(1), zone.MaxSkew)
			require.Equal(t, corev1.ScheduleAnyway, zone.WhenUnsatisfiable)
			require.Nil(t, zone.MinDomains)
			require.Nil(t, zone.NodeTaintsPolicy)
			require.NotNil(t, zone.LabelSelector)
			require.Equal(t, map[string]string{
				"app.kubernetes.io/managed-by": "krane",
				"app.kubernetes.io/component":  "deployment",
			}, zone.LabelSelector.MatchLabels)

			for _, constraint := range dep.Spec.Template.Spec.TopologySpreadConstraints {
				for label, value := range constraint.LabelSelector.MatchLabels {
					require.Equal(t, value, dep.Spec.Template.Labels[label])
				}
			}
		})
	}
}

// TestBuildDeployment_NoCommandUsesImageEntrypoint verifies the safe default:
// when no command override is provided, the container command stays nil so the
// image's ENTRYPOINT/CMD runs.
func TestBuildDeployment_NoCommandUsesImageEntrypoint(t *testing.T) {
	req := fullApplyRequest(t)
	req.Command = nil

	dep := testController().buildDeployment(req, true)
	require.Nil(t, mainContainer(t, dep).Command)
}

// TestBuildDeployment_NoSecretsOmitsEnvFrom verifies that without secrets the
// container has no envFrom mount and the pod uses no dedicated service account.
func TestBuildDeployment_NoSecretsOmitsEnvFrom(t *testing.T) {
	dep := testController().buildDeployment(fullApplyRequest(t), false)
	require.Empty(t, mainContainer(t, dep).EnvFrom)
	require.Empty(t, dep.Spec.Template.Spec.ServiceAccountName)
}

// TestBuildDeployment_GvisorToggle pins both ends: pods are sandboxed by
// default, and disabling it leaves them on the node's default runtime rather
// than naming a RuntimeClass the node may not have.
func TestBuildDeployment_GvisorToggle(t *testing.T) {
	dep := testController().buildDeployment(fullApplyRequest(t), true)
	require.Equal(t, new(runtimeClassGvisor), dep.Spec.Template.Spec.RuntimeClassName)

	c := testController()
	c.disableGvisor = true
	dep = c.buildDeployment(fullApplyRequest(t), true)
	require.Nil(t, dep.Spec.Template.Spec.RuntimeClassName)
}

// TestBuildDeployment_PreStopDrain checks that a pod with the default shutdown
// signal keeps serving after it starts terminating, until frontline has
// reloaded its instance list.
func TestBuildDeployment_PreStopDrain(t *testing.T) {
	for _, signal := range []string{"", "SIGTERM"} {
		t.Run("signal="+signal, func(t *testing.T) {
			req := fullApplyRequest(t)
			req.ShutdownSignal = signal

			c := mainContainer(t, testController().buildDeployment(req, false))
			require.NotNil(t, c.Lifecycle)
			require.NotNil(t, c.Lifecycle.PreStop)
			require.Equal(t, &corev1.SleepAction{Seconds: preStopDrainSeconds}, c.Lifecycle.PreStop.Sleep)
			require.Nil(t, c.Lifecycle.PreStop.Exec)
		})
	}
}

func TestPreStopHandler(t *testing.T) {
	sleepOnly := &corev1.LifecycleHandler{Sleep: &corev1.SleepAction{Seconds: preStopDrainSeconds}}
	for _, tt := range []struct {
		signal string
		want   *corev1.LifecycleHandler
	}{
		{"", sleepOnly},
		{"SIGTERM", sleepOnly},
		{"SIGQUIT; rm -rf /", sleepOnly},
		{"SIGINT", &corev1.LifecycleHandler{Exec: &corev1.ExecAction{Command: []string{
			"sh", "-c", "sleep 15; kill -s INT 1; while kill -0 1 2>/dev/null; do sleep 1; done",
		}}}},
		{"SIGQUIT", &corev1.LifecycleHandler{Exec: &corev1.ExecAction{Command: []string{
			"sh", "-c", "sleep 15; kill -s QUIT 1; while kill -0 1 2>/dev/null; do sleep 1; done",
		}}}},
	} {
		t.Run(tt.signal, func(t *testing.T) {
			require.Equal(t, tt.want, preStopHandler(tt.signal))
		})
	}
}
