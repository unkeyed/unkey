package deployment

import (
	"context"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/conc"
	"github.com/unkeyed/unkey/pkg/logger"
)

type pendingRemoval struct {
	namespace string
	name      string
	retryAt   time.Time
	delay     time.Duration
}

func (c *Controller) trackRemoval(req *ctrlv1.DeleteDeployment) {
	c.removalMu.Lock()
	defer c.removalMu.Unlock()

	if c.pendingRemovals == nil {
		c.pendingRemovals = make(map[string]pendingRemoval)
	}
	if _, exists := c.pendingRemovals[req.GetDeploymentId()]; !exists {
		c.pendingRemovals[req.GetDeploymentId()] = pendingRemoval{
			namespace: req.GetK8SNamespace(),
			name:      req.GetK8SName(),
			retryAt:   time.Now().Add(time.Second),
			delay:     time.Second,
		}
	}
}

func (c *Controller) forgetRemoval(deploymentID string) {
	c.removalMu.Lock()
	defer c.removalMu.Unlock()
	delete(c.pendingRemovals, deploymentID)
}

func (c *Controller) runRemovalRetryLoop(ctx context.Context) {
	c.runResyncLoop(ctx, time.Second, func() {
		if ctx.Err() != nil {
			return
		}
		conc.ForEach(ctx, c.dueRemovals(time.Now()), func(ctx context.Context, hint *ctrlv1.DeleteDeployment) {
			if err := c.ReconcileDeployment(ctx, hint); err != nil {
				logger.Error("unable to retry deployment removal", "deployment_id", hint.GetDeploymentId(), "error", err.Error())
			}
		})
	})
}

func (c *Controller) dueRemovals(now time.Time) []ctrlv1.DeleteDeployment {
	c.removalMu.Lock()
	defer c.removalMu.Unlock()

	var due []ctrlv1.DeleteDeployment
	for deploymentID, removal := range c.pendingRemovals {
		if now.Before(removal.retryAt) {
			continue
		}
		due = append(due, ctrlv1.DeleteDeployment{
			DeploymentId: deploymentID,
			K8SNamespace: removal.namespace,
			K8SName:      removal.name,
		})
		removal.delay = min(removal.delay*2, 30*time.Second)
		removal.retryAt = now.Add(removal.delay)
		c.pendingRemovals[deploymentID] = removal
	}
	return due
}
