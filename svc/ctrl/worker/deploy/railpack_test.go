package deploy

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateShellEnvKeys(t *testing.T) {
	require.NoError(t, validateShellEnvKeys(map[string]string{
		"ZETA":     "1",
		"ALPHA":    "2",
		"_PRIVATE": "3",
		"MIX_3d":   "4",
	}))

	for _, invalid := range []string{"MY-KEY", "my.key", "3LEADING", "SP ACE", `IN"JECT`, "NEW\nLINE"} {
		t.Run(invalid, func(t *testing.T) {
			require.Error(t, validateShellEnvKeys(map[string]string{invalid: "v"}))
		})
	}
}

func TestBuildRailpackPrepareDockerfile(t *testing.T) {
	dockerfile, err := buildRailpackPrepareDockerfile(
		"ghcr.io/unkeyed/railpack-frontend:v0.27.0-unkey.1",
		"ghcr.io/railwayapp/railpack-builder:mise-2026.6.1",
		[]string{"BAR", "FOO"},
		nil,
		"hash123",
	)
	require.NoError(t, err)

	require.True(t, strings.HasPrefix(dockerfile, "# syntax=docker/dockerfile:1\n"))
	require.Contains(t, dockerfile, "FROM ghcr.io/unkeyed/railpack-frontend:v0.27.0-unkey.1 AS railpack")
	require.Contains(t, dockerfile, "FROM ghcr.io/railwayapp/railpack-builder:mise-2026.6.1 AS prepare")
	require.Contains(t, dockerfile, "COPY --from=railpack /railpack /usr/local/bin/railpack")

	// The RUN block is asserted verbatim: a malformed line continuation would
	// only surface as a build failure on the remote machine.
	require.Contains(t, dockerfile, `RUN --mount=type=bind,target=/workspace,readonly \
    --mount=type=cache,target=/tmp/railpack \
    --mount=type=secret,id=BAR,env=BAR \
    --mount=type=secret,id=FOO,env=FOO \
    RAILPACK_SECRETS_HASH=hash123 /usr/local/bin/railpack prepare /workspace --plan-out /railpack-plan.json --env BAR="$BAR" --env FOO="$FOO"`)

	require.Contains(t, dockerfile, "FROM scratch\nCOPY --from=prepare /railpack-plan.json /")

	// Secret values must never appear in the Dockerfile, and without env vars
	// there must be no secret machinery at all.
	bare, err := buildRailpackPrepareDockerfile("frontend:v1", "builder:v1", nil, nil, "")
	require.NoError(t, err)
	require.NotContains(t, bare, "--mount=type=secret")
	require.NotContains(t, bare, "RAILPACK_SECRETS_HASH")

	// Railpack command overrides are mounted as secrets and passed via --env:
	// railpack reads its environment only from --env and interprets RAILPACK_*
	// keys there as config. The fixed names — never the values — appear in the
	// Dockerfile.
	withConfig, err := buildRailpackPrepareDockerfile(
		"frontend:v1", "builder:v1",
		[]string{"FOO"},
		[]string{"RAILPACK_BUILD_CMD", "RAILPACK_INSTALL_CMD"},
		"hash999",
	)
	require.NoError(t, err)
	require.Contains(t, withConfig, `RUN --mount=type=bind,target=/workspace,readonly \
    --mount=type=cache,target=/tmp/railpack \
    --mount=type=secret,id=FOO,env=FOO \
    --mount=type=secret,id=RAILPACK_BUILD_CMD,env=RAILPACK_BUILD_CMD \
    --mount=type=secret,id=RAILPACK_INSTALL_CMD,env=RAILPACK_INSTALL_CMD \
    RAILPACK_SECRETS_HASH=hash999 /usr/local/bin/railpack prepare /workspace --plan-out /railpack-plan.json --env FOO="$FOO" --env RAILPACK_BUILD_CMD="$RAILPACK_BUILD_CMD" --env RAILPACK_INSTALL_CMD="$RAILPACK_INSTALL_CMD"`)

	// Config-only (no env vars): mounts the secret and passes it via --env.
	configOnly, err := buildRailpackPrepareDockerfile(
		"frontend:v1", "builder:v1",
		nil,
		[]string{"RAILPACK_BUILD_CMD"},
		"hash000",
	)
	require.NoError(t, err)
	require.Contains(t, configOnly, `RUN --mount=type=bind,target=/workspace,readonly \
    --mount=type=cache,target=/tmp/railpack \
    --mount=type=secret,id=RAILPACK_BUILD_CMD,env=RAILPACK_BUILD_CMD \
    RAILPACK_SECRETS_HASH=hash000 /usr/local/bin/railpack prepare /workspace --plan-out /railpack-plan.json --env RAILPACK_BUILD_CMD="$RAILPACK_BUILD_CMD"`)
}

func TestRailpackConfigVars(t *testing.T) {
	require.Empty(t, railpackConfigVars(gitBuildParams{}))

	cfg := railpackConfigVars(gitBuildParams{
		BuildCommand:   "pnpm --filter api build",
		InstallCommand: "pnpm install --filter api",
	})
	require.Equal(t, "pnpm --filter api build", cfg[railpackBuildCmdEnv])
	require.Equal(t, "pnpm install --filter api", cfg[railpackInstallCmdEnv])

	// Only the build command set: install is absent, not empty.
	cfg = railpackConfigVars(gitBuildParams{BuildCommand: "go build ./..."})
	require.Equal(t, "go build ./...", cfg[railpackBuildCmdEnv])
	require.NotContains(t, cfg, railpackInstallCmdEnv)
}

func TestHashRailpackPrepareInputs(t *testing.T) {
	require.Empty(t, hashRailpackPrepareInputs(nil, nil))

	// Changing a command override changes the hash so a cached plan for the same
	// commit is not reused.
	base := hashRailpackPrepareInputs(map[string]string{"FOO": "bar"}, nil)
	withCmd := hashRailpackPrepareInputs(map[string]string{"FOO": "bar"}, map[string]string{railpackBuildCmdEnv: "x"})
	changedCmd := hashRailpackPrepareInputs(map[string]string{"FOO": "bar"}, map[string]string{railpackBuildCmdEnv: "y"})
	require.NotEqual(t, base, withCmd)
	require.NotEqual(t, withCmd, changedCmd)
}

func TestRailpackSecrets(t *testing.T) {
	secrets := railpackSecrets("gh-token", map[string]string{"FOO": "bar"})
	require.Equal(t, []byte("gh-token"), secrets[gitAuthTokenSecretID])
	require.Equal(t, []byte("bar"), secrets["FOO"])

	// Unauthenticated mode (public repos) must not register an empty token.
	secrets = railpackSecrets("", map[string]string{"FOO": "bar"})
	require.NotContains(t, secrets, gitAuthTokenSecretID)
	require.Len(t, secrets, 1)
}

func TestValidateRailpackPlan(t *testing.T) {
	dir := t.TempDir()
	planPath := filepath.Join(dir, railpackPlanFilename)

	require.Error(t, validateRailpackPlan(planPath), "missing plan must fail")

	require.NoError(t, os.WriteFile(planPath, []byte(""), 0o644))
	require.Error(t, validateRailpackPlan(planPath), "empty plan must fail")

	require.NoError(t, os.WriteFile(planPath, []byte("not json"), 0o644))
	require.Error(t, validateRailpackPlan(planPath), "invalid JSON must fail")

	require.NoError(t, os.WriteFile(planPath, []byte(`{"steps":{}}`), 0o644))
	require.NoError(t, validateRailpackPlan(planPath))
}

func TestIsTransientSolveError(t *testing.T) {
	transient := errors.New(`failed to copy: httpReadSeeker: failed open: unexpected status code https://ghcr.io/v2/railwayapp/railpack-builder/blobs/sha256:3caa1a: 502 Bad Gateway`)
	require.True(t, isTransientSolveError(transient))

	for _, terminal := range []string{
		`process "/bin/sh -c npm run build" did not complete successfully: exit code: 1`,
		`failed to solve: dockerfile parse error on line 3`,
		`failed to read dockerfile: open Dockerfile: no such file or directory`,
	} {
		require.False(t, isTransientSolveError(errors.New(terminal)), terminal)
	}
}
