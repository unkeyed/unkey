---
name: verify-unkey
description: "Drive the Unkey CLI the way a user does, in an isolated home directory, and capture command proof. Use when a change touches build/cli, cmd/, or user-facing unkey commands and the dashboard cluster is not what you need to prove."
---

# Verify the Unkey CLI

The user-facing surface this skill drives is the `unkey` CLI (`build/cli`). People use it to sign in, list projects, and verify keys. The dashboard at `http://localhost:3000` (`mise run dev`) and the HTTP API are separate surfaces. This skill does not start them. `mise run dev` needs Docker and Minikube. Do not treat a CLI result as proof of the dashboard.

Two launches must not share a home directory. `launch` refuses while a current run exists. Call `cleanup` first. Credential commands are isolated only when `secret-tool` is absent, because a Linux keyring is shared by every home directory. When doctor reports `credential_commands: refused`, do not drive `whoami`, `keys verify`, or `projects list`. Drive `version` and `--help` only.

## Launch

From the repo root, with mise on `PATH` and Go 1.27.3 installed (`./dev/install-mise`, then `mise install go`):

```bash
.cursor/skills/verify-unkey/scripts/verify-unkey launch
```

Launch builds `./build/cli` into the run directory and is ready when it prints `ready: unkey version -> <version>`. A local build prints `dev`. That line is the ready signal. The process exits. There is no server to wait for.

## Doctor

Run this before every drive, and again whenever a command looks wrong:

```bash
.cursor/skills/verify-unkey/scripts/verify-unkey doctor
```

Doctor passes only when the launched binary is still the current binary, its version line matches the launch, stderr is empty, and the home directory is the one this run created. Read `keyring` and `credential_commands` before any command that looks up credentials.

## Drive

The harness is `.cursor/skills/verify-unkey/scripts/verify-unkey`. It sets `HOME` to the run directory and unsets `UNKEY_ROOT_KEY`, `UNKEY_CONFIG`, `UNKEY_API_BASE_URL`, `UNKEY_API_URL`, and `UNKEY_OUTPUT`. Invoke the CLI through `run`, not by calling a binary on the operator's `PATH`.

```bash
.cursor/skills/verify-unkey/scripts/verify-unkey run -- version
.cursor/skills/verify-unkey/scripts/verify-unkey run --capture <name> -- whoami --json
```

`--capture` writes `stdout.txt`, `stderr.txt`, `exit.txt`, and `argv.txt` under `.cursor/skills/verify-unkey/evidence/<name>/`. The feature map in `features/` is the source of which commands to run. A proof that uses one entry point is incomplete when that feature lists others.

## Evidence

Capture the command, stdout, stderr, and exit code. For a command that should change nothing on disk, also record a second look at the home directory (`sessions.json` absent). Do not call an internal Go test as a substitute for the CLI. Do not point `--api-url` at a fake unless the feature says so. Signing in (`unkey login`) opens a browser against the dashboard and is outside this map. There is no dry-run flag on these commands. `version` and `--help` do not touch the network. `whoami`, `keys verify`, and `projects list` in this map stop before an API call when no credential exists. Confirm that by an empty stdout on the failing resource commands and by the absence of `sessions.json`.

## Cleanup

```bash
.cursor/skills/verify-unkey/scripts/verify-unkey cleanup
```

Cleanup deletes the current run directory, including its binary and home. It does not delete `.cursor/skills/verify-unkey/evidence/`. After cleanup, doctor must fail because no current run exists. Evidence files must still be present.

## Helpers

`.cursor/skills/verify-unkey/scripts/verify-unkey` is the only helper. Subcommands are `launch`, `doctor`, `run`, and `cleanup`. `run` takes optional `--capture <name>` and then `--` followed by the `unkey` arguments.
