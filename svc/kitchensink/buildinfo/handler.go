// Package buildinfo exposes a string injected at link time. It verifies
// that release pipelines can bake values (version, commit SHA,
// timestamps) into the binary and read them back at runtime.
//
// Contrast with the env probe: env reads the process environment at
// request time; buildinfo reads a constant the linker wrote into the
// binary. The pipeline contract being exercised here is "build-time
// variable → ldflags -X → package var", not "runtime env → os.Getenv".
// In the Unkey builder the value originates from the project variables
// mounted at /run/secrets/.env during the build; see the Dockerfile.
package buildinfo

import (
	"net/http"

	"github.com/unkeyed/unkey/svc/kitchensink/internal/httpx"
)

// Version is overwritten at link time. Override with:
//
//	go build -ldflags "-X 'github.com/unkeyed/unkey/svc/kitchensink/buildinfo.Version=<value>'"
//
// In Bazel the same wiring lives in x_defs on the kitchensink go_binary.
// "unset" is the sentinel returned when no override was passed, so the
// probe still serves a response during local `go run` sessions.
var Version = "unset"

// Handler returns the build-time Version as JSON. Registered by main.go.
func Handler(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"version": Version})
}
