package repofetch

import (
	batchv1 "k8s.io/api/batch/v1"
)

// JobStatus represents the status of a fetch job.
type JobStatus string

const (
	// JobStatusUnknown indicates the job status could not be determined.
	JobStatusUnknown JobStatus = "unknown"

	// JobStatusPending indicates the job has been created but not started.
	JobStatusPending JobStatus = "pending"

	// JobStatusRunning indicates the job is currently executing.
	JobStatusRunning JobStatus = "running"

	// JobStatusSucceeded indicates the job completed successfully.
	JobStatusSucceeded JobStatus = "succeeded"

	// JobStatusFailed indicates the job failed.
	JobStatusFailed JobStatus = "failed"
)

// IsTerminal returns true if the status is a terminal state (succeeded or failed).
func (s JobStatus) IsTerminal() bool {
	return s == JobStatusSucceeded || s == JobStatusFailed
}

func statusFromJob(job *batchv1.Job) JobStatus {
	if job == nil {
		return JobStatusUnknown
	}

	for _, cond := range job.Status.Conditions {
		if cond.Type == batchv1.JobComplete && cond.Status == "True" {
			return JobStatusSucceeded
		}
		if cond.Type == batchv1.JobFailed && cond.Status == "True" {
			return JobStatusFailed
		}
	}

	if job.Status.Active > 0 {
		return JobStatusRunning
	}

	return JobStatusPending
}
