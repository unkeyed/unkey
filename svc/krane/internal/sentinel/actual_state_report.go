package sentinel

import (
	"context"
	"math/rand/v2"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/conc"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// runActualStateReportLoop starts a Kubernetes watch for sentinel Deployments
// and reports actual state changes back to the control plane in real-time.
//
// The watch filters for resources with the "managed-by: krane" and "component: sentinel"
// labels. When a Deployment's status changes, the method evaluates health via
// [determineHealth] and reports it to the control plane.
//
// Events are processed concurrently via [conc.Sem] so that a slow RPC for one
// sentinel does not block reporting for others.
//
// The initial watch must succeed for the controller to start. After that the
// goroutine automatically reconnects with jittered backoff (1-5s) when the
// watch disconnects or times out.
func (c *Controller) runActualStateReportLoop(ctx context.Context) error {
	w, err := c.watchSentinels(ctx)
	if err != nil {
		return err
	}

	go func() {
		sem := conc.NewSem(conc.DefaultConcurrency)
		defer sem.Wait()

		for {
			c.drainSentinelWatch(ctx, w, sem)

			if ctx.Err() != nil {
				return
			}

			backoff := time.Second + time.Millisecond*time.Duration(rand.Float64()*4000)
			logger.Warn("sentinel watch: disconnected, reconnecting", "backoff", backoff)
			time.Sleep(backoff)

			var watchErr error
			w, watchErr = c.watchSentinels(ctx)
			if watchErr != nil {
				logger.Error("sentinel watch: unable to re-establish watch", "error", watchErr.Error())
				continue
			}
		}
	}()

	return nil
}

// watchSentinels creates a new Kubernetes watch for krane-managed sentinel Deployments.
func (c *Controller) watchSentinels(ctx context.Context) (watch.Interface, error) {
	return c.clientSet.AppsV1().Deployments(NamespaceSentinel).Watch(ctx, metav1.ListOptions{
		LabelSelector: labels.New().
			ManagedByKrane().
			ComponentSentinel().
			ToString(),
	})
}

// drainSentinelWatch processes events from a sentinel Deployment watch until
// the channel closes or the context is cancelled. Reports are dispatched
// concurrently through sem so a slow RPC doesn't block other events.
func (c *Controller) drainSentinelWatch(ctx context.Context, w watch.Interface, sem *conc.Sem) {
	for event := range w.ResultChan() {
		switch event.Type {
		case watch.Error:
			logger.Error("error watching sentinel", "event", event.Object)
		case watch.Bookmark:
		case watch.Added, watch.Modified:
			sentinel, ok := event.Object.(*appsv1.Deployment)
			if !ok {
				logger.Error("unable to cast object to deployment")
				continue
			}

			sem.Go(ctx, func(ctx context.Context) {
				if _, err := c.reportSentinelState(ctx, sentinel); err != nil {
					logger.Error("sentinel watch: unable to report status", "error", err.Error(), "name", sentinel.Name)
				}
			})
		case watch.Deleted:
			sentinel, ok := event.Object.(*appsv1.Deployment)
			if !ok {
				logger.Error("unable to cast object to deployment")
				continue
			}

			sem.Go(ctx, func(ctx context.Context) {
				logger.Info("sentinel deleted", "name", sentinel.Name)
				_, err := c.reportIfChanged(ctx, &ctrlv1.ReportSentinelStatusRequest{
					K8SName:           sentinel.Name,
					AvailableReplicas: 0,
					Health:            ctrlv1.Health_HEALTH_UNHEALTHY,
				})
				if err != nil {
					logger.Error("error reporting sentinel status", "error", err.Error())
				}
			})
		}
	}
}

// reportSentinelState computes and reports the health of a sentinel Deployment.
// Returns (true, nil) when a report was sent, (false, nil) when dedup skipped
// it, or (false, err) on RPC failure.
func (c *Controller) reportSentinelState(ctx context.Context, sentinel *appsv1.Deployment) (bool, error) {
	health := determineHealth(sentinel)
	sentinelID := sentinel.Labels[labels.LabelKeySentinelID]
	reported, err := c.reportIfChanged(ctx, &ctrlv1.ReportSentinelStatusRequest{
		K8SName:           sentinel.Name,
		AvailableReplicas: sentinel.Status.AvailableReplicas,
		Health:            health,
		SentinelId:        sentinelID,
		RunningImage:      convergedImage(sentinel),
	})
	if err != nil {
		return false, err
	}
	if reported {
		logger.Info("sentinel added/modified: reported", "name", sentinel.Name)
	}
	return reported, nil
}
