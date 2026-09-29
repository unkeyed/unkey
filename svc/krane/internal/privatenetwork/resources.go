package privatenetwork

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"strconv"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/deploy/appbinding"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func bindingResourceName(app *ctrlv1.PrivateNetworkApp) string {
	return resourceName("unkey-pn-binding", app.GetBindingId()+"/"+app.GetCallerDeploymentId())
}

func (r *Reconciler) cleanup(ctx context.Context, services *corev1.ServiceList, bindings *corev1.ConfigMapList, desiredServices, desiredBindings map[string]struct{}) error {
	for i := range bindings.Items {
		item := &bindings.Items[i]
		if _, ok := desiredBindings[item.Namespace+"/"+item.Name]; ok || !owned(item.Labels) {
			continue
		}
		if err := r.client.CoreV1().ConfigMaps(item.Namespace).Delete(ctx, item.Name, deleteOptions(item)); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete obsolete private network binding %s/%s: %w", item.Namespace, item.Name, err)
		}
	}

	for i := range services.Items {
		item := &services.Items[i]
		if _, ok := desiredServices[item.Namespace+"/"+item.Name]; ok || !owned(item.Labels) {
			continue
		}
		deadline, ok := retirementDeadline(item)
		if !ok {
			updated := item.DeepCopy()
			updated.Annotations = maps.Clone(item.Annotations)
			if updated.Annotations == nil {
				updated.Annotations = make(map[string]string)
			}
			updated.Annotations[appbinding.RetireAfterAnnotation] = r.clock().Add(appbinding.ReplacementOverlap).UTC().Format(time.RFC3339Nano)
			if _, err := r.client.CoreV1().Services(item.Namespace).Update(ctx, updated, metav1.UpdateOptions{FieldManager: fieldManager}); err != nil {
				return fmt.Errorf("schedule obsolete private network Service %s/%s retirement: %w", item.Namespace, item.Name, err)
			}
			continue
		}
		if r.clock().Before(deadline) {
			continue
		}
		if err := r.client.CoreV1().Services(item.Namespace).Delete(ctx, item.Name, deleteOptions(item)); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete obsolete private network Service %s/%s: %w", item.Namespace, item.Name, err)
		}
	}
	return nil
}

func (r *Reconciler) clock() time.Time {
	if r.now != nil {
		return r.now()
	}
	return time.Now()
}

func retirementDeadline(service *corev1.Service) (time.Time, bool) {
	raw := service.Annotations[appbinding.RetireAfterAnnotation]
	if raw == "" {
		return time.Time{}, false
	}
	deadline, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, true
	}
	return deadline, true
}

func owned(l map[string]string) bool {
	return l[labels.LabelKeyManagedBy] == "krane" &&
		l[labels.LabelKeyComponent] == component &&
		l[labels.LabelKeyWorkspaceID] != "" &&
		l[labels.LabelKeyProjectID] != "" &&
		l[labels.LabelKeyAppID] != ""
}

func ownedByApp(l map[string]string, app *ctrlv1.PrivateNetworkApp) bool {
	return owned(l) &&
		l[labels.LabelKeyWorkspaceID] == app.GetWorkspaceId() &&
		l[labels.LabelKeyProjectID] == app.GetProjectId() &&
		l[labels.LabelKeyAppID] == app.GetAppId()
}

func discoveryName(deployment string, port int32) string {
	return resourceName("unkey-pn-v3", deployment+"/"+strconv.Itoa(int(port)))
}

func deleteOptions(object metav1.Object) metav1.DeleteOptions {
	uid := object.GetUID()
	version := object.GetResourceVersion()
	return metav1.DeleteOptions{Preconditions: &metav1.Preconditions{UID: &uid, ResourceVersion: &version}}
}

func resourceName(prefix, id string) string {
	sum := sha256.Sum256([]byte(id))
	return prefix + "-" + hex.EncodeToString(sum[:16])
}
