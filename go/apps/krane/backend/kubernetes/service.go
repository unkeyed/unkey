package kubernetes

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/unkeyed/unkey/go/gen/proto/krane/v1/kranev1connect"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/repeat"
)

type k8s struct {
	logger    logging.Logger
	clientset *kubernetes.Clientset
	kranev1connect.UnimplementedDeploymentServiceHandler
}

func New(logger logging.Logger) (*k8s, error) {
	// Create in-cluster config
	inClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(inClusterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	repeat.Every(time.Minute, func() {
		ctx := context.Background()
		deployments, err := clientset.AppsV1().Deployments("unkey").List(ctx, metav1.ListOptions{
			LabelSelector: "unkey.managed.by=krane",
		})

		if err != nil {
			logger.Error("failed to list deployments",
				"error", err.Error(),
			)

			return
		}

		for _, deployment := range deployments.Items {

			if time.Since(deployment.GetCreationTimestamp().Time) > (2 * time.Hour) {
				logger.Info("deployment is old and will be deleted",
					"name", deployment.Name,
				)

				err = clientset.AppsV1().Deployments("unkey").Delete(ctx, deployment.Name, metav1.DeleteOptions{

					PropagationPolicy: ptr.P(metav1.DeletePropagationBackground),
				})
				if err != nil {
					logger.Error("failed to delete deployment",
						"error", err.Error(),
						"uid", string(deployment.GetUID()),
						"name", deployment.Name,
					)
				}
			}
		}

	})

	return &k8s{
		logger:    logger,
		clientset: clientset,
	}, nil
}
