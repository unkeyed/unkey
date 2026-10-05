# Create key

Keys for a keyspace are created from the Keys page. The dialog posts to the API and then shows the secret once.

## Sub-features

- Keys page heading `Keys` at `/{workspaceSlug}/apis/{apiId}/keys/{keyAuthId}`.
- Button `Create key`. It stays disabled until `api.queryApiKeyDetails` returns `keyAuth`.
- Dialog title `Create key`, subtitle `Create a custom API key with your own settings`.
- Sections `General Setup`, `Ratelimit`, `Credits`, `Expiration`, `Permissions`, `Metadata`.
- Submit button `Create key` inside the dialog (`form="new-key-form"`).
- Success state heading `Key Created` and the sentence `You've successfully generated a new API key.`
- Closing warns with `You won't see this secret key again!` and confirm button `Close anyway`.

## How to get to it (user POV)

From a drivable instance, open Keyspaces, then open `Local API` (seed id `api_local`). In the section nav, activate `Keys`. The seeded keyspace id is `ks_local`, so the URL is `/local/apis/api_local/keys/ks_local`.

The `Keys` sidebar item stays disabled until the keyspace id resolves. Wait until the link is enabled rather than clicking the API requests URL again.

## Driving it with the browser

Requires verdict `drivable`.

1. Open `http://127.0.0.1:3000/local/apis/api_local/keys/ks_local`.
2. Confirm the heading `Keys`.
3. Activate the button named `Create key` (the page header control, not a disabled one).
4. Confirm the dialog named `Create key` and the section `General Setup`.
5. Leave the defaults unless the form reports an error, then activate the dialog's `Create key` button.
6. Confirm the text `Key Created`. Do not copy the secret into git. Dismiss with `Close anyway` only after the evidence screenshot hides the secret value.

An empty key list shows `No API Keys Found`. After a successful create, the new key row is the side effect once the dialog closes and the list reloads.

## Gotchas

- Two controls share the name `Create key`: the header button and the dialog submit. The dialog must be open before the submit is the right target.
- If the header button is disabled, key auth has not loaded. Wait. Do not invent a `ks_` id.
- The success dialog's title is visually hidden and also repeated as visible text `Key Created`.
- This path needs the API on port 7070. A toast `Failed to Create Key` ends the attempt.
