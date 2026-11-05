# Show the version

Show the version prints the build version of this `unkey` binary and nothing else.

## Sub-features

- `version-command` prints the version from `unkey version`.
- `version-flag` prints the same version from `unkey --version`.

## How to get to it (user POV)

- Run `unkey version`.
- Run `unkey --version`.

## Driving it with verify-unkey

Preconditions:

- Doctor prints `status: ready`.
- The ready line from launch names the version to expect. A local build prints `dev`.

- **Command.** Run `verify-unkey run --capture version-command -- version`. Exit is `0`. Stdout is that version followed by a newline. Stderr is empty.
- **Flag.** Run `verify-unkey run --capture version-flag -- --version`. Exit is `0`. Stdout matches the command form. Stderr is empty.
- **Proof.** The two stdout files are identical, and neither contains `sessions.json`.

## Gotchas

- A local `go build` prints `dev`, not a release semver. Compare against the launch ready line.
- `unkey version` does not read the home directory. A failure there is a bad binary, not a missing login.
