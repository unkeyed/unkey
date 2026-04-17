package sentinel

import (
	"context"
	"math/rand/v2"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// runPodWatchLoop watches sentinel pods and re-evaluates deployment health
// whenever a pod event fires.
//
// The Deployment-level watch (runActualStateReportLoop) may not fire when a pod
// crashes or enters ErrImagePull because the Deployment status fields don't
// always change. Watching pods directly gives us immediate visibility into
// individual pod failures.
//
// On any pod event, we find the owning Deployment, re-read its status, and
// report health via determineHealth.
func (c *Controller) runPodWatchLoop(ctx context.Context) error {
	w, err := c.watchSentinelPods(ctx)
	if err != nil {
		return err
	}

	go func() {
		for {
			c.drainSentinelPodWatch(ctx, w)

			if ctx.Err() != nil {
				return
			}

			backoff := time.Second + time.Millisecond*time.Duration(rand.Float64()*4000)
			logger.Warn("sentinel pod watch: disconnected, reconnecting", "backoff", backoff)
			time.Sleep(backoff)

			var watchErr error
			w, watchErr = c.watchSentinelPods(ctx)
			if watchErr != nil {
				logger.Error("sentinel pod watch: unable to re-establish watch", "error", watchErr.Error())
				continue
			}
		}
	}()

	return nil
}

// watchSentinelPods creates a new Kubernetes watch for pods in the sentinel namespace.
func (c *Controller) watchSentinelPods(ctx context.Context) (watch.Interface, error) {
	return c.clientSet.CoreV1().Pods(NamespaceSentinel).Watch(ctx, metav1.ListOptions{
		LabelSelector: labels.New().
			ManagedByKrane().
			ComponentSentinel().
			ToString(),
	})
}

// drainSentinelPodWatch processes events from a sentinel pod watch.
// On any pod change, it finds the owning Deployment and reports its health.
func (c *Controller) drainSentinelPodWatch(ctx context.Context, w watch.Interface) {
	for event := range w.ResultChan() {
		switch event.Type {
		case watch.Error:
			logger.Error("sentinel pod watch: error event", "event", event.Object)
		case watch.Bookmark:
		case watch.Added, watch.Modified, watch.Deleted:
			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				logger.Error("sentinel pod watch: unable to cast object to pod")
				continue
			}

			// Find the owning Deployment by traversing Pod → ReplicaSet → Deployment.
			deploymentName := owningDeployment(pod)
			if deploymentName == "" {
				continue
			}

			deployment, err := c.clientSet.AppsV1().Deployments(NamespaceSentinel).
				Get(ctx, deploymentName, metav1.GetOptions{})
			if err != nil {
				logger.Error("sentinel pod watch: unable to get deployment",
					"deployment", deploymentName, "pod", pod.Name, "error", err.Error())
				continue
			}

			health := determineHealth(deployment)
			sentinelID := deployment.Labels[labels.LabelKeySentinelID]
			if err := c.reportSentinelStatus(ctx, &ctrlv1.ReportSentinelStatusRequest{
				K8SName:           deployment.Name,
				AvailableReplicas: deployment.Status.AvailableReplicas,
				Health:            health,
				SentinelId:        sentinelID,
				RunningImage:      convergedImage(deployment),
			}); err != nil {
				logger.Error("sentinel pod watch: unable to report status",
					"deployment", deploymentName, "error", err.Error())
			}
		}
	}
}

// owningDeployment traverses a pod's owner references to find the name of the
// Deployment that ultimately owns it (Pod → ReplicaSet → Deployment).
// Returns empty string if the ownership chain can't be determined from labels.
func owningDeployment(pod *corev1.Pod) string {
	// Sentinel pods have a deterministic label set by the Deployment template.
	// The Deployment name matches the k8s_name which is on the pod labels
	// via the pod template. We can get it from the ReplicaSet owner ref
	// and trim the hash suffix, but it's simpler to use the fact that
	// sentinel Deployments set pod-template-hash on the ReplicaSet name.
	//
	// Walk: Pod → ownerRef(ReplicaSet) → ReplicaSet name has format "{deployment}-{hash}"
	// But we don't have the RS object here. Instead, use the app label
	// or just strip the pod hash.
	//
	// Actually the simplest: sentinel pods inherit Deployment labels including
	// the sentinel ID. But we need the Deployment *name* to do a Get().
	// The Deployment name is the k8s_name which isn't in pod labels.
	//
	// So we walk the owner chain: Pod → ReplicaSet name → strip hash → Deployment name.
	for _, ref := range pod.OwnerReferences {
		if ref.Kind == "ReplicaSet" && ref.Controller != nil && *ref.Controller {
			// ReplicaSet name format: "{deployment-name}-{pod-template-hash}"
			// The hash is the last segment after the final hyphen.
			rsName := ref.Name
			for i := len(rsName) - 1; i >= 0; i-- {
				if rsName[i] == '-' {
					return rsName[:i]
				}
			}
			return rsName
		}
	}
	return ""
}
