# Unkey CLI verification map

This directory is the maintained source for verifying the user-facing behavior of the `unkey` CLI. Read this index before driving the CLI, then use the matching feature file as the recipe.

## Baseline preconditions

- Launch with `.cursor/skills/verify-unkey/scripts/verify-unkey launch` and require a `ready: unkey version ->` line.
- Run doctor and require `status: ready`.
- The run home is the only home the CLI may see. Never export the operator's `HOME`.
- Unset `UNKEY_ROOT_KEY`, `UNKEY_CONFIG`, `UNKEY_API_BASE_URL`, `UNKEY_API_URL`, and `UNKEY_OUTPUT`. The harness does this.
- Drive credential commands only when doctor prints `credential_commands: allowed`.
- Never drive a binary that was not built by the current launch.

## Driving conventions

- Start every recipe from the baseline state unless its preconditions say otherwise.
- Run every CLI action through `verify-unkey run`.
- Treat every command as literal. Keep flags and the `--` separator unchanged.
- A non-zero exit is the expected result for a signed-out credential command. Do not retry it with a root key to force a success.
- Record the feature file and the entry point with every captured artifact.
- Report an unreachable path with the command you attempted and the unmet precondition.
- Do not report a skipped entry point as verified through a different path.

## Proof and skip reporting

- Capture the command, stdout, stderr, and exit code.
- For a refusal, capture the refusal text and a second look that shows no session file was written.
- `whoami` writes `~/.unkey/sessions.lock` in the run home even when no session exists. That lock is not a saved session. `sessions.json` must stay absent.
- Resource commands such as `keys verify` and `projects list` print errors on stderr and leave stdout empty. `--json` does not turn those errors into JSON. `whoami` is the exception: it prints a report on stdout and the error on stderr.

## Feature entry contract

Each feature file starts with an H1 and one paragraph, then exactly four H2 sections: `Sub-features`, `How to get to it (user POV)`, `Driving it with verify-unkey`, and `Gotchas`.

## Features

- [Show the version](./version.md) covers `unkey version` and `unkey --version`.
- [Show command help](./help.md) covers `unkey --help`.
- [Show signed-out status](./auth-status.md) covers `unkey whoami` and `unkey auth status`.
- [Refuse key verification without credentials](./keys-verify.md) covers `unkey keys verify`.
- [Refuse project listing without credentials](./projects-list.md) covers `unkey projects list`.
