package deployment

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/pkg/conc"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
)

type workload struct {
	namespace    string
	k8sName      string
	deploymentID string
	selector     *metav1.LabelSelector
}

func workloadFromDeployment(d *appsv1.Deployment) (workload, error) {
	deploymentID, ok := labels.GetDeploymentID(d.Labels)
	if !ok {
		return workload{}, fmt.Errorf("deployment %s/%s has no deployment ID label", d.Namespace, d.Name)
	}

	return workload{namespace: d.Namespace, k8sName: d.Name, deploymentID: deploymentID, selector: d.Spec.Selector}, nil
}

func workloadFromReplicaSet(rs *appsv1.ReplicaSet) (workload, error) {
	deploymentID, ok := labels.GetDeploymentID(rs.Labels)
	if !ok {
		return workload{}, fmt.Errorf("replicaset %s/%s has no deployment ID label", rs.Namespace, rs.Name)
	}

	return workload{namespace: rs.Namespace, k8sName: rs.Name, deploymentID: deploymentID, selector: rs.Spec.Selector}, nil
}

func isLegacyReplicaSet(rs *appsv1.ReplicaSet) bool {
	return metav1.GetControllerOf(rs) == nil
}

func (c *Controller) workloadForReplicaSet(rs *appsv1.ReplicaSet) (workload, error) {
	owner := metav1.GetControllerOf(rs)
	if owner == nil {
		return workloadFromReplicaSet(rs)
	}
	if owner.Kind != "Deployment" {
		return workload{}, fmt.Errorf("replicaset %s/%s is controlled by unexpected kind %s", rs.Namespace, rs.Name, owner.Kind)
	}

	d, err := c.deployments.Deployments(rs.Namespace).Get(owner.Name)
	if err != nil {
		return workload{}, err
	}
	if d.UID != owner.UID {
		return workload{}, fmt.Errorf("deployment %s/%s does not match owner of replicaset %s", d.Namespace, d.Name, rs.Name)
	}

	return workloadFromDeployment(d)
}

func managedWorkloadSelector() string {
	return labels.New().ManagedByKrane().ComponentDeployment().ToString()
}

func (c *Controller) forEachWorkload(ctx context.Context, fn func(ctx context.Context, w workload)) {
	deployments, err := c.deployments.List(k8slabels.Everything())
	if err != nil {
		logger.Error("unable to list deployments", "error", err.Error())
	}
	conc.ForEach(ctx, deployments, func(ctx context.Context, d **appsv1.Deployment) {
		w, err := workloadFromDeployment(*d)
		if err != nil {
			logger.Error("skipping deployment", "error", err.Error())
			return
		}
		fn(ctx, w)
	})

	replicaSets, err := c.replicaSets.List(k8slabels.Everything())
	if err != nil {
		logger.Error("unable to list replicaSets", "error", err.Error())
		return
	}
	conc.ForEach(ctx, replicaSets, func(ctx context.Context, rs **appsv1.ReplicaSet) {
		if !isLegacyReplicaSet(*rs) {
			return
		}
		w, err := workloadFromReplicaSet(*rs)
		if err != nil {
			logger.Error("skipping replicaset", "error", err.Error())
			return
		}
		fn(ctx, w)
	})
}
