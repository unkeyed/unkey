#!/usr/bin/env bash
# Install repo Cursor rules into ~/.cursor/rules (idempotent).
# Run after clone or on a fresh cloud agent VM: ./dev/install-cursor-rules.sh
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
src="${repo_root}/.cursor/rules"
dest="${HOME}/.cursor/rules"

if [ ! -d "$src" ]; then
  echo "No .cursor/rules in repo at $src" >&2
  exit 1
fi

mkdir -p "$dest"
shopt -s nullglob
files=("$src"/*)
shopt -u nullglob

if [ "${#files[@]}" -eq 0 ]; then
  echo "No rule files in $src" >&2
  exit 1
fi

for f in "${files[@]}"; do
  name="$(basename "$f")"
  cp "$f" "${dest}/${name}"
  echo "Installed ${dest}/${name}"
done

echo "Cursor rules ready."
