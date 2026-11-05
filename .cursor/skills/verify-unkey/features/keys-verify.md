# Refuse key verification without credentials

Refuse key verification stops a signed-out user before any key is sent to the API. A missing key is a usage error. A key without a credential is a credentials error. Neither path stores a session.

## Sub-features

- `verify-missing-key` rejects `unkey keys verify` when `--key` is omitted.
- `verify-no-credential` rejects `unkey keys verify --key=sk_example` when no credential exists.

## How to get to it (user POV)

- Run `unkey keys verify`.
- Run `unkey keys verify --key=sk_example`.

## Driving it with verify-unkey

Preconditions:

- Doctor prints `status: ready` and `credential_commands: allowed`.
- No `UNKEY_ROOT_KEY` is set. The harness unsets it.

- **Missing key.** Run `verify-unkey run --capture verify-missing-key -- keys verify`. Exit is `1`. Stdout is empty. Stderr is `required flag missing: key`.
- **No credential.** Run `verify-unkey run --capture verify-no-credential -- keys verify --key=sk_example`. Exit is `1`. Stdout is empty. Stderr contains `Error: unkey keys verify failed.` and `no credentials provided`. Stderr does not contain `sk_example`.
- **No saved session.** `$RUN_HOME/.unkey/sessions.json` does not exist.
- **Proof.** Keep both capture directories. Empty stdout plus the credentials sentence is the refusal. Do not follow it with a live API call.

## Gotchas

- `--json` does not change either refusal. The missing-key error stays plain text on stderr, and the credentials error does too.
- A terminal can open an interactive prompt when required flags are missing. This harness is not a terminal. If stderr waits with no exit, the session is interactive and the result is invalid. Non-interactive runs must exit with `required flag missing: key`.
- If doctor prints `credential_commands: refused`, stop and report that precondition. Do not pass `--root-key` to get past it.
