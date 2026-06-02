#!/usr/bin/env bash
set -euo pipefail

version="${RELEASE_VERSION:-unknown}"
revision="$(git rev-parse HEAD 2>/dev/null || echo unknown)"
build_time="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

echo "STABLE_VERSION ${version}"
echo "STABLE_GIT_COMMIT ${revision}"
echo "BUILD_TIMESTAMP ${build_time}"
