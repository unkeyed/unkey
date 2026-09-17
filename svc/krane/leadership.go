package krane

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

func runWithLeadership(ctx context.Context, client kubernetes.Interface, namespace, identity string, run func(context.Context)) error {
	for ctx.Err() == nil {
		lock := &resourcelock.LeaseLock{
			LeaseMeta:  metav1.ObjectMeta{Name: "krane", Namespace: namespace},
			Client:     client.CoordinationV1(),
			LockConfig: resourcelock.ResourceLockConfig{Identity: identity, EventRecorder: nil},
			Labels:     nil,
		}
		var mu sync.Mutex
		var workers sync.WaitGroup
		stopped := false
		elector, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
			Lock:          lock,
			LeaseDuration: 15 * time.Second,
			RenewDeadline: 10 * time.Second,
			RetryPeriod:   2 * time.Second,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(leaderCtx context.Context) {
					mu.Lock()
					if stopped {
						mu.Unlock()
						return
					}
					workers.Add(1)
					mu.Unlock()
					defer workers.Done()
					logger.Info("krane leadership acquired", "identity", identity)
					run(leaderCtx)
				},
				OnStoppedLeading: func() {},
				OnNewLeader:      nil,
			},
			WatchDog:        nil,
			ReleaseOnCancel: false,
			Name:            "krane",
			Coordinated:     false,
		})
		if err != nil {
			return fmt.Errorf("configure krane leader election: %w", err)
		}
		elector.Run(ctx)
		// client-go does not join OnStartedLeading, which can start after Run returns.
		mu.Lock()
		stopped = true
		mu.Unlock()
		workers.Wait()
		releaseLeadership(ctx, lock)
	}
	return nil
}

func releaseLeadership(ctx context.Context, lock resourcelock.Interface) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	record, _, err := lock.Get(ctx)
	if err != nil {
		logger.Warn("unable to read lease for release", "error", err)
		return
	}
	if record.HolderIdentity != lock.Identity() {
		return
	}
	record.HolderIdentity = ""
	if err := lock.Update(ctx, *record); err != nil {
		logger.Warn("unable to release lease; waiting for expiry", "error", err)
		return
	}
	logger.Info("krane leadership released", "identity", lock.Identity())
}
