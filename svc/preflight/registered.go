package preflight

// Blank imports register every probe into the probes.Registry via
// their init() functions. Keeping the list here (rather than in the
// probes/ package root) means:
//
//  1. Callers that import svc/preflight get a fully-populated
//     registry. The binary (cmd/preflight) and probemanifest both
//     hit this path.
//  2. Individual probe unit tests can import their own package
//     directly without pulling in every other probe's init(),
//     keeping cold-start test cost low.
//
// When adding a new probe, add its subpackage here and regenerate
// svc/preflight/probes/MANIFEST.txt via:
//
//	go run ./cmd/preflight/probemanifest --write
import (
	_ "github.com/unkeyed/unkey/svc/preflight/probes/common/clickhouse_connectivity"
	_ "github.com/unkeyed/unkey/svc/preflight/probes/common/create_deployment"
	_ "github.com/unkeyed/unkey/svc/preflight/probes/common/github_webhook"
	_ "github.com/unkeyed/unkey/svc/preflight/probes/common/noop"
	_ "github.com/unkeyed/unkey/svc/preflight/probes/common/request_logs"
	_ "github.com/unkeyed/unkey/svc/preflight/probes/realinfra/git_push"
	_ "github.com/unkeyed/unkey/svc/preflight/probes/realinfra/krane_contract"
	_ "github.com/unkeyed/unkey/svc/preflight/probes/realinfra/rollback"
)
