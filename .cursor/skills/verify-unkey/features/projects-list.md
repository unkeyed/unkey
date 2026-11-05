# Refuse project listing without credentials

Refuse project listing stops a signed-out user from calling the projects API. The command exits with an error and does not print a project list.

## Sub-features

- `projects-no-credential` rejects `unkey projects list` when no credential exists.

## How to get to it (user POV)

- Run `unkey projects list`.
- The root help example `unkey projects list` is the same command.

## Driving it with verify-unkey

Preconditions:

- Doctor prints `status: ready` and `credential_commands: allowed`.

- **List.** Run `verify-unkey run --capture projects-no-credential -- projects list`. Exit is `1`. Stdout is empty. Stderr contains `Error: unkey projects list failed.` and `no credentials provided`.
- **JSON flag.** Run `verify-unkey run --capture projects-no-credential-json -- projects list --json`. Exit is `1`. Stdout is empty. Stderr contains the same two sentences. It is not a JSON document.
- **No saved session.** `$RUN_HOME/.unkey/sessions.json` does not exist.
- **Proof.** Keep both capture directories.

## Gotchas

- If doctor prints `credential_commands: refused`, stop and report that precondition.
- A successful project list requires `unkey login` against a running API. That path is not this feature. Do not mark this feature passed because help printed the example.
