package diskupload_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/unkeyed/unkey/svc/preflight/core"
	"github.com/unkeyed/unkey/svc/preflight/runner/diskupload"
)

func TestUpload_WritesArtifactsUnderRunIDAndProbe(t *testing.T) {
	root := t.TempDir()
	u := diskupload.New(root)

	err := u.Upload(context.Background(), "run-123", "git_push", []core.Artifact{
		{Name: "deployment.json", ContentType: "application/json", Body: []byte(`{"id":"dep_1"}`)},
		{Name: "steps.json", ContentType: "application/json", Body: []byte(`[]`)},
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	depBytes, err := os.ReadFile(filepath.Join(root, "run-123", "git_push", "deployment.json"))
	if err != nil {
		t.Fatalf("read deployment.json: %v", err)
	}
	if string(depBytes) != `{"id":"dep_1"}` {
		t.Fatalf("deployment.json body = %q; want {\"id\":\"dep_1\"}", string(depBytes))
	}
	if _, err := os.Stat(filepath.Join(root, "run-123", "git_push", "steps.json")); err != nil {
		t.Fatalf("steps.json missing: %v", err)
	}
}

func TestUpload_NoArtifacts_DoesNotCreateDirectory(t *testing.T) {
	root := t.TempDir()
	u := diskupload.New(root)

	if err := u.Upload(context.Background(), "run-456", "noop", nil); err != nil {
		t.Fatalf("Upload: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "run-456")); !os.IsNotExist(err) {
		t.Fatalf("expected no run-456 dir; got err=%v", err)
	}
}
