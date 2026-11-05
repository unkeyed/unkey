# Show command help

Show command help lists the workspace commands a signed-in user can run, including the `whoami` alias, and exits successfully without a credential.

## Sub-features

- `help-root` prints the root command list from `unkey --help`.

## How to get to it (user POV)

- Run `unkey --help`.
- Run `unkey help`.

## Driving it with verify-unkey

Preconditions:

- Doctor prints `status: ready`.

- **Root help.** Run `verify-unkey run --capture help-root -- --help`. Exit is `0`. Stdout contains `unkey - Manage your Unkey workspace`, `whoami → unkey auth status`, and `unkey keys list --api-id=api_123 --json`. Stderr is empty.
- **Help command.** Run `verify-unkey run --capture help-command -- help`. Exit is `0`. Stdout contains the same three strings.
- **Proof.** Both captures include `keys` and `projects`. Stderr files are empty.

## Gotchas

- Help text wraps the description across lines. Match the phrases above, not a single wrapped line.
- `unkey help` and `unkey --help` are both entry points. Passing only one does not cover the other.
