package core

// Artifact is one diagnostic blob uploaded on failure. The Runner
// writes these to s3://<bucket>/<run-id>/<probe>/<name> and links them
// from the alert.
//
// Keep artifacts small. A probe's own debug bundle is useful; a full
// pod log tarball is not. Think kubectl-describe output, a JSON
// snapshot of the failing row, the last 50 lines of a service log.
type Artifact struct {
	Name        string
	ContentType string
	Body        []byte
}
