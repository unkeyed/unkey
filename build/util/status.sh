#!/usr/bin/env bash
# Bazel workspace status command (wired via --workspace_status_command in .bazelrc).
# Emits STABLE_* keys that Bazel substitutes into x_defs to stamp pkg/buildinfo
# (Version, Revision, BuildTime) into release binaries.
# See: https://bazel.build/docs/user-manual#workspace-status
set -euo pipefail

version="${RELEASE_VERSION:-unknown}"
revision="$(git rev-parse HEAD 2>/dev/null || echo unknown)"
build_time="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

echo "STABLE_VERSION ${version}"
echo "STABLE_GIT_COMMIT ${revision}"
echo "BUILD_TIMESTAMP ${build_time}"
