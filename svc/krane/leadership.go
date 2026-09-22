package krane

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

// leadershipProbeInterval bounds how stale the "why is krane not leading"
// signal can be. Every probe failure is recorded as it happens; the ticker
// only decides how often that state is written to the log.
const leadershipProbeInterval = 30 * time.Second

func runWithLeadership(ctx context.Context, client kubernetes.Interface, namespace, identity string, run func(context.Context)) error {
	for ctx.Err() == nil {
		lock := &resourcelock.LeaseLock{
			LeaseMeta:  metav1.ObjectMeta{Name: "krane", Namespace: namespace},
			Client:     client.CoordinationV1(),
			LockConfig: resourcelock.ResourceLockConfig{Identity: identity, EventRecorder: nil},
			Labels:     nil,
		}

		// client-go reports lease probe failures only through klog, and
		// elector.Run returns only when ctx is done, so without this the
		// reason a krane never becomes leader is invisible in the app log.
		probes := &leaseProbeRecorder{Interface: lock}

		var mu sync.Mutex
		var workers sync.WaitGroup
		stopped := false
		var leading atomic.Bool

		elector, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
			Lock:          probes,
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

					leading.Store(true)
					logger.Info("krane leadership acquired", "identity", identity)
					run(leaderCtx)
				},
				OnStoppedLeading: func() { leading.Store(false) },
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

		electorCtx, cancelElector := context.WithCancel(ctx)
		probesDone := make(chan struct{})
		go func() {
			defer close(probesDone)
			reportLeaseProbeFailures(electorCtx, identity, &leading, probes)
		}()

		elector.Run(electorCtx)
		cancelElector()
		<-probesDone

		// client-go does not join OnStartedLeading, which can start after Run returns.
		mu.Lock()
		stopped = true
		mu.Unlock()

		workers.Wait()
		releaseLeadership(ctx, lock)
	}

	return nil
}

// reportLeaseProbeFailures writes the leadership acquisition failure state on a
// fixed interval while this instance is not leading. A healthy standby that is
// waiting on the active leader records no probe error and stays quiet, so a
// line here means the lease could not be read or written, not merely that
// another replica holds it.
func reportLeaseProbeFailures(ctx context.Context, identity string, leading *atomic.Bool, probes *leaseProbeRecorder) {
	ticker := time.NewTicker(leadershipProbeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		if leading.Load() {
			continue
		}

		err := probes.LastError()
		if err == nil || apierrors.IsAlreadyExists(err) || apierrors.IsConflict(err) {
			continue
		}

		logger.Warn("krane leadership not acquired",
			"identity", identity,
			"error", err,
		)
	}
}

// leaseProbeRecorder wraps the leader election lock so the most recent lease
// probe result is available to [reportLeaseProbeFailures]. client-go discards
// these errors into klog, leaving no application-queryable trace of a 403 or an
// unreachable API server on the lease.
type leaseProbeRecorder struct {
	resourcelock.Interface

	mu      sync.Mutex
	lastErr error
}

func (r *leaseProbeRecorder) Get(ctx context.Context) (*resourcelock.LeaderElectionRecord, []byte, error) {
	record, raw, err := r.Interface.Get(ctx)
	r.record(err)
	return record, raw, err
}

func (r *leaseProbeRecorder) Create(ctx context.Context, record resourcelock.LeaderElectionRecord) error {
	err := r.Interface.Create(ctx, record)
	r.record(err)
	return err
}

func (r *leaseProbeRecorder) Update(ctx context.Context, record resourcelock.LeaderElectionRecord) error {
	err := r.Interface.Update(ctx, record)
	r.record(err)
	return err
}

func (r *leaseProbeRecorder) record(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastErr = err
}

func (r *leaseProbeRecorder) LastError() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastErr
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
