package deployment

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReconcileRevision_RejectsOlderOperations(t *testing.T) {
	controller := &Controller{}
	state := ""

	require.NoError(t, controller.reconcileRevision("dep_1", 1, func() error {
		state = "deleted"
		return nil
	}))
	require.NoError(t, controller.reconcileRevision("dep_1", 0, func() error {
		state = "running"
		return nil
	}))

	require.Equal(t, "deleted", state)
}

func TestReconcileRevision_RejectsOlderConfiguration(t *testing.T) {
	controller := &Controller{}
	configuration := ""

	require.NoError(t, controller.reconcileRevision("dep_1", 4, func() error {
		configuration = "new"
		return nil
	}))
	require.NoError(t, controller.reconcileRevision("dep_1", 3, func() error {
		configuration = "old"
		return nil
	}))

	require.Equal(t, "new", configuration)
}

func TestReconcileRevision_InitialZeroAndEqualReplayApply(t *testing.T) {
	controller := &Controller{}
	applications := 0

	for range 2 {
		require.NoError(t, controller.reconcileRevision("dep_1", 0, func() error {
			applications++
			return nil
		}))
	}

	require.Equal(t, 2, applications)
}

func TestReconcileRevision_FailureDoesNotAdvanceRevision(t *testing.T) {
	controller := &Controller{}
	wantErr := errors.New("apply failed")

	require.ErrorIs(t, controller.reconcileRevision("dep_1", 2, func() error {
		return wantErr
	}), wantErr)
	applied := false
	require.NoError(t, controller.reconcileRevision("dep_1", 1, func() error {
		applied = true
		return nil
	}))

	require.True(t, applied)
}

func TestReconcileRevision_SerializesEachDeploymentIndependently(t *testing.T) {
	controller := &Controller{}
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondStarted := make(chan struct{})
	otherStarted := make(chan struct{})
	errorResults := make(chan error, 3)
	var waitGroup sync.WaitGroup
	waitGroup.Add(3)

	go func() {
		defer waitGroup.Done()
		errorResults <- controller.reconcileRevision("dep_1", 0, func() error {
			close(firstStarted)
			<-releaseFirst
			return nil
		})
	}()
	<-firstStarted

	go func() {
		defer waitGroup.Done()
		errorResults <- controller.reconcileRevision("dep_1", 1, func() error {
			close(secondStarted)
			return nil
		})
	}()
	go func() {
		defer waitGroup.Done()
		errorResults <- controller.reconcileRevision("dep_2", 0, func() error {
			close(otherStarted)
			return nil
		})
	}()

	<-otherStarted
	select {
	case <-secondStarted:
		t.Fatal("operations for one deployment ran concurrently")
	default:
	}
	close(releaseFirst)
	waitGroup.Wait()
	close(errorResults)
	for err := range errorResults {
		require.NoError(t, err)
	}
}
