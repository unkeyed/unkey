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

func crashLooping(reason string, exitCode int32, finishedAt int64) dbtype.ContainerStatus {
	status := exited(reason, exitCode, finishedAt)
	status.Waiting = &dbtype.WaitingState{Reason: "CrashLoopBackOff"}
	return status
}

func pendingInstances(statuses ...dbtype.ContainerStatus) []db.ListDeploymentInstanceStatusesByIdsRow {
	instances := make([]db.ListDeploymentInstanceStatusesByIdsRow, len(statuses))
	for i, status := range statuses {
		instances[i] = db.ListDeploymentInstanceStatusesByIdsRow{
			DeploymentID:    "",
			Status:          db.InstancesStatusPending,
			ContainerStatus: status,
		}
	}
	return instances
}

func TestCrashLoopIssue(t *testing.T) {
	crashLoopingWithoutExit := dbtype.ContainerStatus{
		RestartCount:         3,
		LastTerminationState: nil,
		Waiting:              &dbtype.WaitingState{Reason: "CrashLoopBackOff"},
	}

	cases := []struct {
		name    string
		status  dbtype.ContainerStatus
		reason  *openapi.DeploymentExitReason
		message string
	}{
		{
			name:    "out of memory keeps its reason and advice",
			status:  crashLooping("OOMKilled", 137, 1_000),
			reason:  new(openapi.DeploymentExitReasonOutOfMemory),
			message: "Your app used more memory than its limit and keeps stopping, so Unkey waits longer before each restart. Check its memory usage, or raise the limit.",
		},
		{
			name:    "a start failure points at the start command",
			status:  crashLooping("StartError", 128, 1_000),
			reason:  new(openapi.DeploymentExitReasonStartFailed),
			message: "The start command could not run, so Unkey waits longer before each restart. Check your start command and entrypoint.",
		},
		{
			name:    "a generic error explains a known exit code",
			status:  crashLooping("Error", 127, 1_000),
			reason:  new(openapi.DeploymentExitReasonExitedWithError),
			message: "Your app keeps stopping, so Unkey waits longer before each restart. Exit code 127 means a command was not found. Check your logs for the cause.",
		},
		{
			name:    "an unlisted code above 128 names the signal",
			status:  crashLooping("Error", 134, 1_000),
			reason:  new(openapi.DeploymentExitReasonExitedWithError),
			message: "Your app keeps stopping, so Unkey waits longer before each restart. Exit code 134 means the process was killed by signal 6. Check your logs for the cause.",
		},
		{
			name:    "an unknown reason still reports the exit code",
			status:  crashLooping("RunContainerError", 42, 1_000),
			reason:  new(openapi.DeploymentExitReasonUnknown),
			message: "Your app keeps stopping, so Unkey waits longer before each restart. Your app last exited with code 42. Check your logs for the cause.",
		},
		{
			name:    "clean exits say a service must keep running",
			status:  crashLooping("Completed", 0, 1_000),
			reason:  new(openapi.DeploymentExitReasonExitedCleanly),
			message: "Your app keeps stopping, so Unkey waits longer before each restart. Exit code 0 means your app stopped on its own. A service must keep running. Check your logs for the cause.",
		},
		{
			name:    "a crash loop with no recorded exit has no reason or exit code",
			status:  crashLoopingWithoutExit,
			reason:  nil,
			message: "Your app keeps stopping, so Unkey waits longer before each restart. Check your logs for the cause.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := issues(pendingInstances(tc.status))
			require.Len(t, got, 1)
			issue := got[0]
			require.Equal(t, openapi.DeploymentIssueTypeCrashLoop, issue.Type)
			require.Equal(t, tc.message, issue.Message)
			require.NotNil(t, issue.Meta)
			require.Equal(t, tc.reason, issue.Meta.Reason)
			require.Equal(t, new(int64(tc.status.RestartCount)), issue.Meta.RestartCount)
			if term := tc.status.LastTerminationState; term != nil {
				require.Equal(t, new(term.ExitCode), issue.Meta.ExitCode)
				require.Equal(t, new(term.FinishedAt), issue.Time)
			} else {
				require.Nil(t, issue.Meta.ExitCode)
				require.Nil(t, issue.Time)
			}
		})
	}
}

func TestIssuesSelection(t *testing.T) {
	healthy := dbtype.ContainerStatus{RestartCount: 0, LastTerminationState: nil, Waiting: nil}

	t.Run("no issues when no instance is crash looping", func(t *testing.T) {
		require.Empty(t, issues(nil))
		require.Empty(t, issues(pendingInstances(healthy, healthy)))
		require.Empty(t, issues(pendingInstances(exited("OOMKilled", 137, 1_000))))
	})

	t.Run("a waiting reason other than a crash loop is not an issue", func(t *testing.T) {
		creating := dbtype.ContainerStatus{
			RestartCount:         0,
			LastTerminationState: nil,
			Waiting:              &dbtype.WaitingState{Reason: "ContainerCreating"},
		}
		require.Empty(t, issues(pendingInstances(creating)))
	})

	t.Run("a running instance is not crash looping", func(t *testing.T) {
		instances := pendingInstances(crashLooping("Error", 137, 1_000))
		instances[0].Status = db.InstancesStatusRunning
		require.Empty(t, issues(instances))
	})

	t.Run("several crash-looping instances give one issue from the most recent exit", func(t *testing.T) {
		got := issues(pendingInstances(
			crashLooping("Error", 2, 1_000),
			healthy,
			crashLooping("OOMKilled", 137, 2_000),
			crashLooping("Error", 1, 1_500),
		))
		require.Len(t, got, 1)
		require.Equal(t, new(openapi.DeploymentExitReasonOutOfMemory), got[0].Meta.Reason)
	})

	t.Run("a newer exit of a recovered instance does not hide a crash loop", func(t *testing.T) {
		got := issues(pendingInstances(
			crashLooping("OOMKilled", 137, 1_000),
			exited("Error", 1, 2_000),
		))
		require.Len(t, got, 1)
		require.Equal(t, new(openapi.DeploymentExitReasonOutOfMemory), got[0].Meta.Reason)
		require.Equal(t, new(int64(1_000)), got[0].Time)
	})

	t.Run("equal exit times prefer the instance with more restarts", func(t *testing.T) {
		restarted := crashLooping("Error", 1, 1_000)
		restarted.RestartCount = 5
		got := issues(pendingInstances(crashLooping("Error", 2, 1_000), restarted))
		require.Len(t, got, 1)
		require.Equal(t, new(int64(5)), got[0].Meta.RestartCount)
	})
}
