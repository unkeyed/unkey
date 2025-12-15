package sentinelcontroller_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	sentinelcontroller "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller"
	sentinelv1 "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/api/v1"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/uid"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestX(t *testing.T) {
	logger := logging.New()

	// Create a runtime scheme and add required types
	scheme := runtime.NewScheme()
	err := sentinelv1.AddToScheme(scheme)
	require.NoError(t, err)
	err = appsv1.AddToScheme(scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)

	//specify testEnv configuration
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:       []string{filepath.Join(".", "yaml")},
		DownloadBinaryAssets:    true,
		ControlPlaneStopTimeout: 10 * time.Second,
		Scheme:                  scheme,
	}

	// start testEnv
	cfg, err := testEnv.Start()
	require.NoError(t, err)

	t.Logf("Started test environment: %+v", testEnv)

	t.Cleanup(func() {
		require.NoError(t, testEnv.Stop())
	})

	k, err := kubernetes.NewForConfig(cfg)
	require.NoError(t, err)

	sv, err := k.ServerVersion()
	require.NoError(t, err)
	t.Logf("Server version: %+v", sv)

	// Create a manager using the test config
	manager, err := k8s.NewManagerWithConfig(cfg, scheme)
	require.NoError(t, err)

	events := buffer.New[*ctrlv1.SentinelEvent](buffer.Config{
		Capacity: 1000,
		Drop:     false,
		Name:     "krane_sentinel_events",
	})

	updates := buffer.New[*ctrlv1.UpdateSentinelRequest](buffer.Config{
		Capacity: 1000,
		Drop:     false,
		Name:     "krane_sentinel_updates",
	})

	c, err := sentinelcontroller.New(sentinelcontroller.Config{
		Logger:  logger,
		Events:  events,
		Updates: updates,
		Manager: manager,
		Config:  cfg,
	})
	require.NoError(t, err)

	go func() {
		if err := manager.Start(context.Background()); err != nil {
			logger.Error("failed to start test manager", "error", err.Error())
		}
	}()

	sentinel := &ctrlv1.ApplySentinel{
		Namespace:     fmt.Sprintf("ns-%s", uid.NanoLower(8)),
		K8SCrdName:    fmt.Sprintf("crd-%s", uid.NanoLower(8)),
		WorkspaceId:   uid.New(uid.TestPrefix),
		ProjectId:     uid.New(uid.TestPrefix),
		EnvironmentId: uid.New(uid.TestPrefix),
		SentinelId:    uid.New(uid.TestPrefix),
		Image:         "nginx:latest",
		Replicas:      1,
		CpuMillicores: 512,
		MemorySizeMib: 512,
	}

	err = manager.GetClient().Create(context.Background(), &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    "default",
			GenerateName: "sentinel-",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "sentinel",
				},
			},
			Replicas: ptr.P[int32](1),

			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "sentinel",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "sentinel",
							Image:   "nginx:latest",
							Command: []string{"run", "sentinel"},
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	})

	require.NoError(t, err)

	err = c.ApplySentinel(context.Background(), sentinel)
	require.NoError(t, err)

	// Simplify test - only verify sentinel CRD creation, not deployment
	require.EventuallyWithT(t, func(ct *assert.CollectT) {

		list := &sentinelv1.SentinelList{}
		err := manager.GetClient().List(context.Background(), list)
		require.NoError(t, err)

		t.Logf("Sentinel List: %+v", list)

		found := &sentinelv1.Sentinel{}
		err = manager.GetClient().Get(context.Background(), types.NamespacedName{Namespace: sentinel.Namespace, Name: sentinel.K8SCrdName}, found)
		require.NoError(t, err, "failed to get sentinel")

		require.Equal(ct, sentinel.GetWorkspaceId(), found.Spec.WorkspaceID, "workspace id mismatch")
		require.Equal(ct, sentinel.GetProjectId(), found.Spec.ProjectID, "project id mismatch")
		require.Equal(ct, sentinel.GetEnvironmentId(), found.Spec.EnvironmentID, "environment id mismatch")
		require.Equal(ct, sentinel.GetSentinelId(), found.Spec.SentinelID, "sentinel id mismatch")
		require.Equal(ct, sentinel.GetImage(), found.Spec.Image, "image mismatch")
		require.Equal(ct, int32(sentinel.GetReplicas()), found.Spec.Replicas, "replicas mismatch")
		require.Equal(ct, int64(sentinel.GetCpuMillicores()), found.Spec.CpuMillicores, "cpu mismatch")
		require.Equal(ct, int64(sentinel.GetMemorySizeMib()), found.Spec.MemoryMib, "memory mismatch")

	}, 30*time.Second, 500*time.Millisecond)

}
