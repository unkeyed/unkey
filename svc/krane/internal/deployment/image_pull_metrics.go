package deployment

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/repeat"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type pullFailureKey struct {
	deploymentID  string
	projectID     string
	environmentID string
	workspaceID   string
	reason        string
}

// runImagePullMetricsLoop publishes unkey_krane_pod_image_pull_failures by
// listing all krane-managed deployment pods every 30s and counting pods
// whose containers are Waiting with an image-pull reason.
//
// Stale series (deployment recovered, pod deleted, image pull succeeded)
// are removed via DeleteLabelValues so the gauge reflects current reality
// without requiring pod-watch wiring.
func (c *Controller) runImagePullMetricsLoop(ctx context.Context) {
	selector := labels.New().ManagedByKrane().ComponentDeployment().ToString()
	previous := map[pullFailureKey]struct{}{}

	repeat.Every(30*time.Second, func() {
		pods, err := c.clientSet.CoreV1().Pods("").List(ctx, metav1.ListOptions{
			LabelSelector: selector,
		})
		if err != nil {
			logger.Error("image pull metrics: list pods failed", "error", err.Error())
			return
		}

		current := map[pullFailureKey]int{}
		for i := range pods.Items {
			pod := &pods.Items[i]
			l := pod.GetLabels()
			deploymentID := l[labels.LabelKeyDeploymentID]
			if deploymentID == "" {
				continue
			}
			for _, cs := range pod.Status.ContainerStatuses {
				if cs.State.Waiting == nil {
					continue
				}
				if !isImagePullReason(cs.State.Waiting.Reason) {
					continue
				}
				current[pullFailureKey{
					deploymentID:  deploymentID,
					projectID:     l[labels.LabelKeyProjectID],
					environmentID: l[labels.LabelKeyEnvironmentID],
					workspaceID:   l[labels.LabelKeyWorkspaceID],
					reason:        cs.State.Waiting.Reason,
				}]++
			}
		}

		for key, count := range current {
			setPullFailure(key, float64(count))
		}
		for key := range previous {
			if _, still := current[key]; still {
				continue
			}
			deletePullFailure(key)
		}

		next := make(map[pullFailureKey]struct{}, len(current))
		for key := range current {
			next[key] = struct{}{}
		}
		previous = next
	})
}

func setPullFailure(k pullFailureKey, v float64) {
	metrics.PodImagePullFailures.
		WithLabelValues(k.deploymentID, k.projectID, k.environmentID, k.workspaceID, k.reason).
		Set(v)
}

func deletePullFailure(k pullFailureKey) {
	metrics.PodImagePullFailures.
		DeleteLabelValues(k.deploymentID, k.projectID, k.environmentID, k.workspaceID, k.reason)
}

func isImagePullReason(reason string) bool {
	switch reason {
	case "ImagePullBackOff", "ErrImagePull", "InvalidImageName", "ImageInspectError":
		return true
	}
	return false
}
