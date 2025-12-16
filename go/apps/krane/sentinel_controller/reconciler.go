package sentinelcontroller

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	apiv1 "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/api/v1"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"k8s.io/apimachinery/pkg/api/meta"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Definitions to manage status conditions
const (
	// typeAvailableSentinel represents the status of the Deployment reconciliation
	typeAvailableSentinel = "Available"
)

var (
	requeueNow = ctrl.Result{Requeue: true, RequeueAfter: time.Millisecond}
	// requeuePeriodically is used to keep the cluster resoruces in sync with the database in case any change events go missing
	requeuePeriodically = ctrl.Result{Requeue: true, RequeueAfter: 15 * time.Minute}
	requeueNever        = ctrl.Result{Requeue: false, RequeueAfter: 0}
)

type reconciler struct {
	logger  logging.Logger
	client  client.Client
	scheme  *runtime.Scheme
	cluster ctrlv1connect.ClusterServiceClient
}

var _ k8s.Reconciler = (*reconciler)(nil)

// Reconcile handles reconciliation of Sentinel custom resources.
//
// This method implements the controller-runtime reconcile loop for Sentinel resources.
// It fetches the Sentinel resource, creates or updates the underlying
// Kubernetes StatefulSet, and manages resource status and conditions.
//
// The reconciliation process:
//  1. Fetch the Sentinel CRD
//  2. Create or update corresponding StatefulSet
//  3. Update status conditions based on operation results
//  4. Return appropriate requeue timing
//
// Parameters:
//   - ctx: Context for reconciliation operation
//   - req: Reconciliation request with namespace and name
//
// Returns ctrl.Result indicating when to requeue and any error encountered.
func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	crd := &apiv1.Sentinel{} // nolint:exhaustruct
	err := r.client.Get(ctx, req.NamespacedName, crd)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// The sentinel resource is not found, it must be deleted
			return requeueNever, nil
		}
		// Error reading the object - requeue the request.
		r.logger.Error("failed to get sentinel", "error", err)
		return ctrl.Result{}, err
	}

	// Let's just set the status as Unknown when no status is available
	if len(crd.Status.Conditions) == 0 {
		meta.SetStatusCondition(&crd.Status.Conditions, metav1.Condition{
			Type:    typeAvailableSentinel,
			Status:  metav1.ConditionUnknown,
			Reason:  "Reconciling",
			Message: "Starting reconciliation",
		})
		if err = r.client.Status().Update(ctx, crd); err != nil {
			r.logger.Error("Failed to update Sentinel status", "error", err)
			return ctrl.Result{}, err
		}
		return requeueNow, nil
	}

	state, err := r.cluster.GetDesiredSentinelState(ctx, connect.NewRequest(&ctrlv1.GetDesiredSentinelStateRequest{
		SentinelId: crd.Spec.SentinelID,
	}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			err = r.client.Delete(ctx, crd)
			return ctrl.Result{}, err
		}

		r.logger.Error("Failed to pull sentinel state", "error", err)
		return ctrl.Result{}, err
	}

	sentinel := state.Msg

	deployment, err := r.ensureDeploymentExists(ctx, crd, sentinel)
	if err != nil {

		return ctrl.Result{}, err
	}

	svc, err := r.ensureServiceExists(ctx, crd, sentinel)
	if err != nil {

		return ctrl.Result{}, err
	}

	for _, fn := range []stateReconciler{
		r.reconcileImage,
		r.reconcileReplicas,
		//r.reconcileCPU,
		//r.reconcileMemory,
	} {

		requeue, err := fn(ctx, sentinel, deployment, svc)
		if err != nil {
			return ctrl.Result{}, err
		}
		if requeue {
			r.logger.Info("Requeueing due to state change")
			return requeueNow, nil
		}
	}

	return requeuePeriodically, nil

}

func (r *reconciler) ensureDeploymentExists(ctx context.Context, crd *apiv1.Sentinel, sentinel *ctrlv1.GetDesiredSentinelStateResponse) (*appsv1.Deployment, error) {

	name := fmt.Sprintf("%s-dpl", crd.GetName())
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: crd.GetNamespace(),
			Labels: k8s.NewLabels().
				WorkspaceID(sentinel.GetWorkspaceId()).
				ProjectID(sentinel.GetProjectId()).
				EnvironmentID(sentinel.GetEnvironmentId()).
				SentinelID(sentinel.GetSentinelId()).
				ComponentSentinel().
				ManagedByKrane().
				ToMap(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.P(sentinel.GetReplicas()),
			Selector: &metav1.LabelSelector{
				MatchLabels: k8s.NewLabels().SentinelID(sentinel.GetSentinelId()).ToMap(),
			},
			Template: corev1.PodTemplateSpec{

				ObjectMeta: metav1.ObjectMeta{

					Labels: k8s.NewLabels().SentinelID(sentinel.GetSentinelId()).ToMap(),
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyAlways,

					Containers: []corev1.Container{{
						Image:           sentinel.GetImage(),
						Name:            "sentinel",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Args:            []string{"run", "sentinel"},
						Env: []corev1.EnvVar{
							{Name: "UNKEY_WORKSPACE_ID", Value: sentinel.GetWorkspaceId()},
							{Name: "UNKEY_PROJECT_ID", Value: sentinel.GetProjectId()},
							{Name: "UNKEY_ENVIRONMENT_ID", Value: sentinel.GetEnvironmentId()},
							{Name: "UNKEY_SENTINEL_ID", Value: sentinel.GetSentinelId()},
							{Name: "UNKEY_DATABASE_PRIMARY", Value: "unkey:password@tcp(mysql:3306)/unkey?parseTime=true&interpolateParams=true"},
						},

						Ports: []corev1.ContainerPort{{
							ContainerPort: 8040,
							Name:          "sentinel",
						}},

						//Resources: corev1.ResourceRequirements{
						//	// nolint:exhaustive
						//	Limits: corev1.ResourceList{
						//		corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(sentinel.GetCpuMillicores()), resource.BinarySI),
						//		corev1.ResourceMemory: *resource.NewQuantity(int64(sentinel.GetMemoryMib()), resource.BinarySI),
						//	},
						//	// nolint:exhaustive
						//	Requests: corev1.ResourceList{
						//		corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(sentinel.GetCpuMillicores()), resource.BinarySI),
						//		corev1.ResourceMemory: *resource.NewQuantity(int64(sentinel.GetMemoryMib()), resource.BinarySI),
						//	},
						//},
					}},
				},
			},
		},
	}

	// Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err := r.client.Get(ctx, types.NamespacedName{Namespace: crd.GetNamespace(), Name: name}, found)

	if err == nil {

		found.Spec = deployment.Spec
		if err := r.client.Update(ctx, found); err != nil {
			return nil, err
		}

		return found, nil
	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}

	// Set the ownerRef for the Deployment
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := ctrl.SetControllerReference(crd, deployment, r.scheme); err != nil {
		return nil, err
	}

	if err = r.client.Create(ctx, deployment); err != nil {

		return nil, err
	}

	return deployment, nil

}

func (r *reconciler) ensureServiceExists(ctx context.Context, crd *apiv1.Sentinel, sentinel *ctrlv1.GetDesiredSentinelStateResponse) (*corev1.Service, error) {

	name := fmt.Sprintf("%s-svc", crd.GetName())

	found := &corev1.Service{}
	err := r.client.Get(ctx, types.NamespacedName{Namespace: crd.GetNamespace(), Name: name}, found)

	if err == nil {

		return found, nil
	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: crd.GetNamespace(),
			Labels: k8s.NewLabels().
				WorkspaceID(sentinel.GetWorkspaceId()).
				ProjectID(sentinel.GetProjectId()).
				EnvironmentID(sentinel.GetEnvironmentId()).
				SentinelID(sentinel.GetSentinelId()).
				ManagedByKrane().
				ToMap(),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP, // Use ClusterIP for internal communication
			Selector: k8s.NewLabels().SentinelID(sentinel.GetSentinelId()).ToMap(),
			//nolint:exhaustruct
			Ports: []corev1.ServicePort{
				{
					Port:       8040,
					TargetPort: intstr.FromInt(8040),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}

	// Set the ownerRef for the Deployment
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := ctrl.SetControllerReference(crd, svc, r.scheme); err != nil {
		return nil, err
	}

	if err = r.client.Create(ctx, svc); err != nil {
		return nil, err
	}

	return svc, nil

}

// for every spec field in the CRD we write a state reconciler
// returns true if the resource was changed and therefore should be requeued
type stateReconciler func(ctx context.Context, sentinel *ctrlv1.GetDesiredSentinelStateResponse, deployment *appsv1.Deployment, service *corev1.Service) (bool, error)

func (r *reconciler) reconcileImage(ctx context.Context, sentinel *ctrlv1.GetDesiredSentinelStateResponse, deployment *appsv1.Deployment, _ *corev1.Service) (bool, error) {
	r.logger.Info("Reconciling image")
	for i, container := range deployment.Spec.Template.Spec.Containers {
		if container.Image != sentinel.GetImage() {
			deployment.Spec.Template.Spec.Containers[i].Image = sentinel.GetImage()

			err := r.client.Update(ctx, deployment)
			if err != nil {
				return false, err
			}
			return true, nil

		}

	}

	return false, nil

}

func (r *reconciler) reconcileReplicas(ctx context.Context, sentinel *ctrlv1.GetDesiredSentinelStateResponse, deployment *appsv1.Deployment, _ *corev1.Service) (bool, error) {

	r.logger.Info("Reconciling replicas")
	if deployment.Spec.Replicas == nil || *deployment.Spec.Replicas != sentinel.GetReplicas() {
		deployment.Spec.Replicas = ptr.P(sentinel.GetReplicas())

		err := r.client.Update(ctx, deployment)
		if err != nil {
			return false, err
		}
		return true, nil

	}

	return false, nil

}
