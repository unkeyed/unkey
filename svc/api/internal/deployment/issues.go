package deployment

import (
	"cmp"
	"fmt"
	"slices"
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
	2:   "Exit code 2 usually means a wrong command, wrong arguments, or a shell syntax error.",
	126: "Exit code 126 means a command exists but cannot run. Check its permissions.",
	127: "Exit code 127 means a command was not found.",
	137: "Exit code 137 means the process was killed with SIGKILL, for example because it did not stop in time after SIGTERM.",
	139: "Exit code 139 means the process crashed on an invalid memory access.",
	143: "Exit code 143 means the process stopped on SIGTERM, for example after a failed health check.",
}

// issues reports at most one crash loop per deployment, from the crash-looping
// instance with the most recent exit, so one stuck instance in a multi-region
// rollout still surfaces while the others are healthy.
func issues(instances []db.ListDeploymentInstanceStatusesByIdsRow) []openapi.DeploymentIssue {
	crashLooping := slices.DeleteFunc(slices.Clone(instances), func(instance db.ListDeploymentInstanceStatusesByIdsRow) bool {
		return !isCrashLooping(instance)
	})
	if len(crashLooping) == 0 {
		return []openapi.DeploymentIssue{}
	}
	latest := slices.MaxFunc(crashLooping, func(a, b db.ListDeploymentInstanceStatusesByIdsRow) int {
		return cmp.Or(
			cmp.Compare(exitFinishedAt(a.ContainerStatus), exitFinishedAt(b.ContainerStatus)),
			cmp.Compare(a.ContainerStatus.RestartCount, b.ContainerStatus.RestartCount),
		)
	})

	meta := openapi.DeploymentIssueMeta{
		Reason:       nil,
		ExitCode:     nil,
		RestartCount: new(int64(latest.ContainerStatus.RestartCount)),
	}
	issue := openapi.DeploymentIssue{
		Type:    openapi.DeploymentIssueTypeCrashLoop,
		Time:    nil,
		Message: "",
		Meta:    &meta,
	}
	if term := latest.ContainerStatus.LastTerminationState; term != nil {
		meta.Reason = new(cmp.Or(terminationReasons[term.Reason], openapi.DeploymentExitReasonUnknown))
		meta.ExitCode = new(term.ExitCode)
		issue.Time = new(term.FinishedAt)
	}
	issue.Message = crashLoopMessage(meta)
	return []openapi.DeploymentIssue{issue}
}

func crashLoopMessage(meta openapi.DeploymentIssueMeta) string {
	if meta.Reason != nil {
		switch *meta.Reason {
		case openapi.DeploymentExitReasonOutOfMemory:
			return "Your app used more memory than its limit and keeps stopping, so Unkey waits longer before each restart. Check its memory usage, or raise the limit."
		case openapi.DeploymentExitReasonStartFailed:
			return "The start command could not run, so Unkey waits longer before each restart. Check your start command and entrypoint."
		case openapi.DeploymentExitReasonExitedWithError, openapi.DeploymentExitReasonExitedCleanly, openapi.DeploymentExitReasonUnknown:
		}
	}

	sentences := []string{"Your app keeps stopping, so Unkey waits longer before each restart."}
	if exitCode := meta.ExitCode; exitCode != nil {
		switch {
		case exitCodeHints[*exitCode] != "":
			sentences = append(sentences, exitCodeHints[*exitCode])
		case *exitCode > signalExitBase:
			sentences = append(sentences, fmt.Sprintf("Exit code %d means the process was killed by signal %d.", *exitCode, *exitCode-signalExitBase))
		default:
			sentences = append(sentences, fmt.Sprintf("Your app last exited with code %d.", *exitCode))
		}
	}
	return strings.Join(append(sentences, "Check your logs for the cause."), " ")
}

func isCrashLooping(instance db.ListDeploymentInstanceStatusesByIdsRow) bool {
	waiting := instance.ContainerStatus.Waiting
	return instance.Status != db.InstancesStatusRunning && waiting != nil && waiting.Reason == "CrashLoopBackOff"
}

func exitFinishedAt(status dbtype.ContainerStatus) int64 {
	if status.LastTerminationState == nil {
		return noRecordedExit
	}
	return status.LastTerminationState.FinishedAt
}
