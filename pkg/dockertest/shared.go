package dockertest

import (
	"sync"
	"testing"
)

// shared is a process-level singleton container.
// The container starts on first use and lives for the binary's lifetime.
// Each test binary (Bazel test target) gets its own process, so containers
// are naturally isolated between targets.
type shared struct {
	once sync.Once
	ctr  *Container
}

// get returns the shared container, starting it on first call.
// The container is NOT registered for t.Cleanup — it outlives individual tests.
// Orphan cleanup is handled by the "owner=dockertest" label + `make clean-docker-test`.
func (s *shared) get(t *testing.T, cfg containerConfig) *Container {
	t.Helper()

	s.once.Do(func() {
		cfg.SkipCleanup = true
		s.ctr = startContainer(t, cfg)
	})

	return s.ctr
}
