package deploymentcontroller

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	apiv1 "github.com/unkeyed/unkey/go/apps/krane/deployment_controller/api/v1"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/uid"
	// +kubebuilder:scaffold:imports
)

type TestHarness struct {
	t         *testing.T
	ctx       context.Context
	cancel    context.CancelFunc
	testEnv   *envtest.Environment
	cfg       *rest.Config
	k8sClient client.Client
	manager   manager.Manager
	namespace string
	Logger    logging.Logger
}

func NewTestHarness(t *testing.T) *TestHarness {

	logger := logging.New()
	controllerruntime.SetLogger(k8s.CompatibleLogger(logger))

	ctx, cancel := context.WithCancel(context.TODO())

	t.Cleanup(func() {
		cancel()
	})

	require.NoError(t, apiv1.AddToScheme(scheme.Scheme))

	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join(".", "yaml")},
		ErrorIfCRDPathMissing: true,
		DownloadBinaryAssets:  true,
		Scheme:                scheme.Scheme,
	}

	t.Cleanup(func() {
		require.NoError(t, testEnv.Stop())
	})

	// cfg is defined in this file globally.
	cfg, err := testEnv.Start()
	require.NoError(t, err)

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	require.NoError(t, err)

	manager, err := manager.New(cfg, manager.Options{
		Scheme: scheme.Scheme,
		Controller: config.Controller{
			SkipNameValidation: ptr.P(true),
		},
	})
	require.NoError(t, err)

	namespace := uid.DNS1035()

	err = k8sClient.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, k8sClient.Delete(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}))
	})

	h := &TestHarness{
		t:         t,
		ctx:       ctx,
		cancel:    cancel,
		testEnv:   testEnv,
		cfg:       cfg,
		k8sClient: k8sClient,
		manager:   manager,
		namespace: namespace,
		Logger:    logger,
	}

	return h
}

func (h *TestHarness) NewController() *DeploymentController {

	c, err := New(Config{
		Logger:  h.Logger,
		Scheme:  h.k8sClient.Scheme(),
		Client:  h.k8sClient,
		Manager: h.manager,
	})
	require.NoError(h.t, err)

	return c

}
func (h *TestHarness) FullReconcileOrFail(r reconcile.Reconciler, namespace, name string) {

	var result reconcile.Result
	var err error

	for range 100 {
		result, err = r.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: namespace,
				Name:      name,
			},
		})
		if err == nil && result.IsZero() {
			break
		}

	}

	require.NoError(h.t, err)
	require.True(h.t, result.IsZero())

}
