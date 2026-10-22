package deployment

import (
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

const (
	noRecordedExit int64 = -1
	signalExitBase int32 = 128
)

var terminationReasons = map[string]openapi.DeploymentExitReason{
	"OOMKilled":          openapi.DeploymentExitReasonOutOfMemory,
	"Error":              openapi.DeploymentExitReasonExitedWithError,
	"ContainerCannotRun": openapi.DeploymentExitReasonStartFailed,
	"StartError":         openapi.DeploymentExitReasonStartFailed,
	"Completed":          openapi.DeploymentExitReasonExitedCleanly,
}

var exitCodeHints = map[int32]string{
	0:   "Exit code 0 means your app stopped on its own. A service must keep running.",
	1:   "Exit code 1 usually means an unhandled error.",
	2:   "Exit code 2 frequently indicates incorrect command usage, invalid arguments, or shell/syntax errors.",
	126: "Exit code 126 means a command exists but cannot run. Check its permissions.",
	127: "Exit code 127 means a command was not found.",
	137: "Exit code 137 means the process was killed with SIGKILL.",
	139: "Exit code 139 means the process crashed on an invalid memory access.",
	143: "Exit code 143 means the process stopped on SIGTERM. This is normal during a redeploy.",
}

// lastExit picks the most recent exit across the instances of a deployment, so
// one failing instance in a multi-region rollout still surfaces while the others
// are healthy. A crash loop outranks the exit reason because it means the
// instance is stuck, and the exit code still carries the cause. ctrl keeps the
// crash loop on the row after the instance recovers, so it counts only while
// the instance is not running.
func lastExit(instances []db.ListInstanceContainerStatusesByDeploymentIdsRow) *openapi.DeploymentLastExit {
	var latest *dbtype.ContainerStatus
	latestCrashLooping := false
	latestFinishedAt := noRecordedExit
	for i := range instances {
		status := &instances[i].ContainerStatus
		crashLooping := instances[i].Status != db.InstancesStatusRunning &&
			status.Waiting != nil && status.Waiting.Reason == "CrashLoopBackOff"
		if status.LastTerminationState == nil && !crashLooping {
			continue
		}
		finishedAt := noRecordedExit
		if status.LastTerminationState != nil {
			finishedAt = status.LastTerminationState.FinishedAt
		}
		if latest == nil || finishedAt > latestFinishedAt ||
			(finishedAt == latestFinishedAt && status.RestartCount > latest.RestartCount) {
			latest, latestCrashLooping, latestFinishedAt = status, crashLooping, finishedAt
		}
	}
	if latest == nil {
		return nil
	}

	exit := &openapi.DeploymentLastExit{
		Reason:       openapi.DeploymentExitReasonUnknown,
		Message:      "",
		ExitCode:     nil,
		RestartCount: int64(latest.RestartCount),
		FinishedAt:   nil,
	}
	if term := latest.LastTerminationState; term != nil {
		exit.ExitCode = new(term.ExitCode)
		exit.FinishedAt = new(term.FinishedAt)
		if reason, ok := terminationReasons[term.Reason]; ok {
			exit.Reason = reason
		}
	}
	if latestCrashLooping {
		exit.Reason = openapi.DeploymentExitReasonCrashLoop
	}
	exit.Message = exitMessage(exit.Reason, exit.ExitCode)
	return exit
}

func exitMessage(reason openapi.DeploymentExitReason, exitCode *int32) string {
	switch reason {
	case openapi.DeploymentExitReasonOutOfMemory:
		return "Your app used more memory than its limit and was stopped. Check its memory usage, or raise the limit."
	case openapi.DeploymentExitReasonStartFailed:
		return "The start command could not run. Check your start command and entrypoint."
	case openapi.DeploymentExitReasonCrashLoop, openapi.DeploymentExitReasonExitedWithError, openapi.DeploymentExitReasonExitedCleanly, openapi.DeploymentExitReasonUnknown:
	}

	var sentences []string
	if reason == openapi.DeploymentExitReasonCrashLoop {
		sentences = append(sentences, "Your app exits right after it starts, so Unkey waits longer before each restart.")
	}
	switch {
	case exitCode == nil:
		if reason != openapi.DeploymentExitReasonCrashLoop {
			sentences = append(sentences, "Unkey could not tell why your app stopped.")
		}
	case exitCodeHints[*exitCode] != "":
		sentences = append(sentences, exitCodeHints[*exitCode])
	case *exitCode > signalExitBase:
		sentences = append(sentences, fmt.Sprintf("Exit code %d means the process was killed by signal %d.", *exitCode, *exitCode-signalExitBase))
	default:
		sentences = append(sentences, fmt.Sprintf("Your app exited with code %d.", *exitCode))
	}
	return strings.Join(append(sentences, "Check your logs for the cause."), " ")
}
