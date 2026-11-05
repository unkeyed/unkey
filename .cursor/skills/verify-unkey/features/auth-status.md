# Show signed-out status

Show signed-out status tells the user that this machine has no Unkey credential. The human report and the JSON report both say the credential is missing, and the command exits with an error. No browser session is stored.

## Sub-features

- `whoami-human` prints the missing-credential tree from `unkey whoami`.
- `whoami-json` prints the same fact as JSON from `unkey whoami --json`.
- `auth-status` prints the JSON report from `unkey auth status --json`.

## How to get to it (user POV)

- Run `unkey whoami`.
- Run `unkey whoami --json`.
- Run `unkey auth status --json`.

## Driving it with verify-unkey

Preconditions:

- Doctor prints `status: ready` and `credential_commands: allowed`.
- The run home has no `.unkey/sessions.json`.

- **Human.** Run `verify-unkey run --capture whoami-human -- whoami`. Exit is `1`. Stdout contains `Authentication: missing` and `Credential: none`. Stderr contains `no credentials provided` and `unkey login`.
- **JSON alias.** Run `verify-unkey run --capture whoami-json -- whoami --json`. Exit is `1`. Stdout contains `"credential": "none"` and `"authentication": "missing"`. Stderr contains `no credentials provided`.
- **Auth status.** Run `verify-unkey run --capture auth-status -- auth status --json`. Exit is `1`. Stdout matches the JSON alias. Stderr contains `no credentials provided`.
- **No saved session.** After the three commands, `$RUN_HOME/.unkey/sessions.json` does not exist. A `sessions.lock` file may exist. That lock is not a credential.
- **Proof.** Keep the three capture directories. The absence of `sessions.json` is the second view.

## Gotchas

- If doctor prints `credential_commands: refused`, stop. `secret-tool` would read the operator's keyring, which this home directory does not isolate. Report that precondition. Do not unset the keyring or delete the operator's secrets.
- Exit `1` with `credential: none` is the passing result. A root key that makes this exit `0` is a different scenario and does not satisfy this feature.
- JSON is indented and includes `apiUrl` of `https://api.unkey.com` when no API URL was set. Do not require the API to be reachable. This path returns before an API call.
