package deploymentcontroller

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/krane/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (c *DeploymentController) GetRunningDeploymentIds(ctx context.Context) <-chan string {
	deploymentIDs := make(chan string)

	go func() {

		defer close(deploymentIDs)
		cursor := ""
		for {

			statefulsets, err := c.clientset.AppsV1().StatefulSets(k8s.UntrustedNamespace).List(ctx, metav1.ListOptions{
				LabelSelector: labels.FormatLabels(k8s.NewLabels().
					ManagedByKrane().
					ToMap()),
				Continue: cursor,
				Limit:    100,
			})
			if err != nil {
				c.logger.Error("unable to list stateful sets", "error", err.Error())
				return
			}
			cursor = statefulsets.GetContinue()

			for _, sfs := range statefulsets.Items {
				deploymentID, ok := k8s.GetDeploymentID(sfs.GetLabels())

				if !ok {
					c.logger.Warn("skipping non-Deployment sfs", "name", sfs.Name)
					continue
				}
				deploymentIDs <- deploymentID
			}
			if cursor == "" {
				return
			}

		}

	}()
	return deploymentIDs
}
