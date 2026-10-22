package deployment

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func exited(reason string, exitCode int32, finishedAt int64) dbtype.ContainerStatus {
	return dbtype.ContainerStatus{
		RestartCount: 1,
		LastTerminationState: &dbtype.TerminatedState{
			ExitCode:   exitCode,
			Signal:     0,
			Reason:     reason,
			FinishedAt: finishedAt,
		},
		Waiting: nil,
	}
}

func pendingInstances(statuses ...dbtype.ContainerStatus) []db.ListInstanceContainerStatusesByDeploymentIdsRow {
	instances := make([]db.ListInstanceContainerStatusesByDeploymentIdsRow, len(statuses))
	for i, status := range statuses {
		instances[i] = db.ListInstanceContainerStatusesByDeploymentIdsRow{
			DeploymentID:    "",
			Status:          db.InstancesStatusPending,
			ContainerStatus: status,
		}
	}
	return instances
}

func TestLastExitClassification(t *testing.T) {
	crashLooping := exited("Error", 137, 1_000)
	crashLooping.Waiting = &dbtype.WaitingState{Reason: "CrashLoopBackOff"}
	cleanCrashLooping := exited("Completed", 0, 1_000)
	cleanCrashLooping.Waiting = &dbtype.WaitingState{Reason: "CrashLoopBackOff"}
	crashLoopingWithoutExit := dbtype.ContainerStatus{
		RestartCount:         3,
		LastTerminationState: nil,
		Waiting:              &dbtype.WaitingState{Reason: "CrashLoopBackOff"},
	}

	cases := []struct {
		name    string
		status  dbtype.ContainerStatus
		reason  openapi.DeploymentExitReason
		message string
	}{
		{
			name:    "a crash loop wins over the exit reason and explains the exit code",
			status:  crashLooping,
			reason:  openapi.DeploymentExitReasonCrashLoop,
			message: "Your app exits right after it starts, so Unkey waits longer before each restart. Exit code 137 means the process was killed with SIGKILL. Check your logs for the cause.",
		},
		{
			name:    "a crash loop with no recorded exit has no exit code",
			status:  crashLoopingWithoutExit,
			reason:  openapi.DeploymentExitReasonCrashLoop,
			message: "Your app exits right after it starts, so Unkey waits longer before each restart. Check your logs for the cause.",
		},
		{
			name:    "out of memory does not repeat the exit code",
			status:  exited("OOMKilled", 137, 1_000),
			reason:  openapi.DeploymentExitReasonOutOfMemory,
			message: "Your app used more memory than its limit and was stopped. Check its memory usage, or raise the limit.",
		},
		{
			name:    "a generic error explains a known exit code",
			status:  exited("Error", 127, 1_000),
			reason:  openapi.DeploymentExitReasonExitedWithError,
			message: "Exit code 127 means a command was not found. Check your logs for the cause.",
		},
		{
			name:    "an unlisted code above 128 names the signal",
			status:  exited("Error", 134, 1_000),
			reason:  openapi.DeploymentExitReasonExitedWithError,
			message: "Exit code 134 means the process was killed by signal 6. Check your logs for the cause.",
		},
		{
			name:    "an unknown reason still reports the exit code",
			status:  exited("RunContainerError", 42, 1_000),
			reason:  openapi.DeploymentExitReasonUnknown,
			message: "Your app exited with code 42. Check your logs for the cause.",
		},
		{
			name:    "a clean exit says a service must keep running",
			status:  exited("Completed", 0, 1_000),
			reason:  openapi.DeploymentExitReasonExitedCleanly,
			message: "Exit code 0 means your app stopped on its own. A service must keep running. Check your logs for the cause.",
		},
		{
			name:    "a crash loop of clean exits explains exit code 0",
			status:  cleanCrashLooping,
			reason:  openapi.DeploymentExitReasonCrashLoop,
			message: "Your app exits right after it starts, so Unkey waits longer before each restart. Exit code 0 means your app stopped on its own. A service must keep running. Check your logs for the cause.",
		},
		{
			name:    "a start failure points at the start command",
			status:  exited("StartError", 128, 1_000),
			reason:  openapi.DeploymentExitReasonStartFailed,
			message: "The start command could not run. Check your start command and entrypoint.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := lastExit(pendingInstances(tc.status))
			require.NotNil(t, got)
			require.Equal(t, tc.reason, got.Reason)
			require.Equal(t, tc.message, got.Message)
			require.Equal(t, int64(tc.status.RestartCount), got.RestartCount)
			if term := tc.status.LastTerminationState; term != nil {
				require.Equal(t, new(term.ExitCode), got.ExitCode)
				require.Equal(t, new(term.FinishedAt), got.FinishedAt)
			} else {
				require.Nil(t, got.ExitCode)
				require.Nil(t, got.FinishedAt)
			}
		})
	}
}

func TestLastExitSelection(t *testing.T) {
	healthy := dbtype.ContainerStatus{RestartCount: 0, LastTerminationState: nil, Waiting: nil}

	t.Run("no exit when every instance is healthy", func(t *testing.T) {
		require.Nil(t, lastExit(nil))
		require.Nil(t, lastExit(pendingInstances(healthy, healthy)))
	})

	t.Run("the most recent exit across instances wins", func(t *testing.T) {
		got := lastExit(pendingInstances(
			exited("Error", 2, 1_000),
			healthy,
			exited("OOMKilled", 137, 2_000),
			exited("Error", 1, 1_500),
		))
		require.NotNil(t, got)
		require.Equal(t, openapi.DeploymentExitReasonOutOfMemory, got.Reason)
	})

	t.Run("equal exit times prefer the instance with more restarts", func(t *testing.T) {
		restarted := exited("Error", 1, 1_000)
		restarted.RestartCount = 5
		got := lastExit(pendingInstances(exited("Error", 2, 1_000), restarted))
		require.NotNil(t, got)
		require.Equal(t, openapi.DeploymentExitReasonExitedWithError, got.Reason)
		require.Equal(t, int64(5), got.RestartCount)
	})

	t.Run("a waiting reason other than a crash loop is not an exit", func(t *testing.T) {
		creating := dbtype.ContainerStatus{
			RestartCount:         0,
			LastTerminationState: nil,
			Waiting:              &dbtype.WaitingState{Reason: "ContainerCreating"},
		}
		require.Nil(t, lastExit(pendingInstances(creating)))
	})

	t.Run("a running instance reports its last exit, not the crash loop it recovered from", func(t *testing.T) {
		recovered := exited("Error", 137, 1_000)
		recovered.Waiting = &dbtype.WaitingState{Reason: "CrashLoopBackOff"}
		instances := pendingInstances(recovered)
		instances[0].Status = db.InstancesStatusRunning

		got := lastExit(instances)
		require.NotNil(t, got)
		require.Equal(t, openapi.DeploymentExitReasonExitedWithError, got.Reason)
		require.Equal(t, "Exit code 137 means the process was killed with SIGKILL. Check your logs for the cause.", got.Message)

		instances[0].ContainerStatus.LastTerminationState = nil
		require.Nil(t, lastExit(instances))
	})
}
