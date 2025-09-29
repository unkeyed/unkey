package kubernetes

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/repeat"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// autoEvictDeployments removes deployments that are older than the specified TTL.
//
// This function implements automatic cleanup of old deployments to prevent
// resource accumulation in development and testing environments. It runs
// continuously in a background goroutine, scanning for deployments that
// exceed the configured time-to-live threshold.
func (k *k8s) autoEvictDeployments(ttl time.Duration) {

	repeat.Every(time.Minute, func() {
		ctx := context.Background()
		deployments, err := k.clientset.AppsV1().StatefulSets("unkey").List(ctx, metav1.ListOptions{
			LabelSelector: "unkey.managed.by=krane",
		})

		if err != nil {
			k.logger.Error("failed to list deployments",
				"error", err.Error(),
			)

			return
		}

		for _, deployment := range deployments.Items {

			if time.Since(deployment.GetCreationTimestamp().Time) > ttl {
				k.logger.Info("deployment is old and will be deleted",
					"name", deployment.Name,
				)

				err = k.clientset.AppsV1().StatefulSets("unkey").Delete(ctx, deployment.Name, metav1.DeleteOptions{

					PropagationPolicy: ptr.P(metav1.DeletePropagationBackground),
				})
				if err != nil {
					k.logger.Error("failed to delete deployment",
						"error", err.Error(),
						"uid", string(deployment.GetUID()),
						"name", deployment.Name,
					)
				}
			}
		}

	})
}
