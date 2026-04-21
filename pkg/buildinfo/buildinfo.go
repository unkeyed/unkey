// Package buildinfo exposes build metadata that is injected at link time
// via -ldflags -X. Values are populated by the release pipeline (goreleaser).
// Local builds see the defaults.
//
// To emit a Prometheus build_info gauge for a service, call
// [RegisterBuildInfoMetrics] once during service startup, after the prometheus
// registry has been installed via lazy.SetRegistry.
package buildinfo

var (
	// Version is the release version (semver tag, e.g. "v0.1.0").
	Version = "dev"

	// Revision is the full git SHA the binary was built from.
	Revision = "unknown"

	// BuildTime is the RFC3339 UTC timestamp of the build (e.g. "2026-04-16T10:30:45Z").
	BuildTime = "unknown"
)
