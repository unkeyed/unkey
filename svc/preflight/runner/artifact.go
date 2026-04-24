package runner

import (
	"context"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/preflight/core"
)

// ArtifactUploader abstracts the diagnostic-bundle sink. Production
// implementations wrap S3; tests substitute an in-memory fake. The
// Runner only calls Upload on failure, and only when the probe or
// its Diagnose method produced at least one artifact.
type ArtifactUploader interface {
	Upload(ctx context.Context, runID, probeName string, artifacts []core.Artifact) error
}

// upload wraps ArtifactUploader.Upload with a best-effort log-on-error
// so a broken uploader cannot turn a real probe failure into a
// confusing "upload error" at the top of the bundle.
func (r *Runner) upload(ctx context.Context, probeName, runID string, artifacts []core.Artifact) {
	if err := r.artifacts.Upload(ctx, runID, probeName, artifacts); err != nil {
		logger.Error("preflight: artifact upload failed",
			"suite", r.suite,
			"probe", probeName,
			"run_id", runID,
			"error", err.Error(),
		)
	}
}
