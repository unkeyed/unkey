// Package diskupload writes failure artifacts to a local directory.
// It is the default ArtifactUploader for dev runs and the fallback
// when the production S3 uploader is not configured. Layout:
//
//	<root>/<run-id>/<probe-name>/<artifact-name>
//
// The Runner only calls Upload on failure; passing it a Result with
// no artifacts is also safe (no-op, no empty directory created).
package diskupload

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/preflight/core"
)

// Uploader satisfies runner.ArtifactUploader by writing each artifact
// to the local filesystem under Root.
type Uploader struct {
	Root string
}

// New returns an Uploader rooted at root. Root is created lazily on
// the first Upload call so a probe run that never fails leaves no
// directory behind.
func New(root string) *Uploader {
	return &Uploader{Root: root}
}

// Upload writes each artifact as <Root>/<runID>/<probeName>/<a.Name>.
// Returns the first write error; partial writes are not cleaned up
// because they are usually still useful for debugging.
func (u *Uploader) Upload(_ context.Context, runID, probeName string, artifacts []core.Artifact) error {
	if len(artifacts) == 0 {
		return nil
	}

	dir := filepath.Join(u.Root, runID, probeName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("diskupload: mkdir %s: %w", dir, err)
	}

	for _, a := range artifacts {
		path := filepath.Join(dir, a.Name)
		if err := os.WriteFile(path, a.Body, 0o600); err != nil {
			return fmt.Errorf("diskupload: write %s: %w", path, err)
		}
	}

	logger.Info("preflight: artifacts written",
		"dir", dir,
		"count", len(artifacts),
	)
	return nil
}
