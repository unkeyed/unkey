# Keyspaces

The API list is titled Keyspaces. Creating one calls the Unkey API and then opens that API's requests page.

## Sub-features

- List at `/{workspaceSlug}/apis` with heading `Keyspaces`.
- Empty state heading `Create your first keyspace`.
- Search miss heading `No keyspaces found`.
- Opener button `Create keyspace`. Dialog title `Create keyspace`. Name field label `Name`. Submit button `Create Keyspace`.
- Success toast `Your keyspace has been created`, then navigation to `/{workspaceSlug}/apis/{apiId}`.

## How to get to it (user POV)

Sign in through the local landing only when doctor says `drivable`. In the sidebar, activate `Keyspaces (APIs)`. The URL is `/local/apis` after the default seed. The header reads `Keyspaces`.

Direct URL: `http://127.0.0.1:3000/local/apis`. `http://127.0.0.1:3000/~/apis` redirects to the same path once a workspace row exists for the session org.

## Driving it with the browser

Requires verdict `drivable`.

1. Activate the link named `Keyspaces (APIs)`.
2. Confirm the heading `Keyspaces`. The default seed already inserts `Local API` (`api_local`), so the empty state is absent on a fresh seed.
3. Activate the button named `Create keyspace`.
4. In the dialog named `Create keyspace`, fill the textbox named `Name` with a value of 3 to 50 characters, for example `verify-keyspace`.
5. Activate the button named `Create Keyspace`.
6. Confirm the toast `Your keyspace has been created` and that the URL matches `/local/apis/api_` plus a new id. The requests page for that API is the resulting state.

Creating a keyspace calls `apis.createApi` through the dashboard's Unkey client, which needs the API process on port 7070 and `UNKEY_ROOT_KEY` when the procedure is rate limited. If the toast is `Failed to Create Keyspace`, record the description and stop. Do not retry against another host.

## Gotchas

- The opener label is `Create keyspace`. The submit label is `Create Keyspace`.
- `?new=true` on the list opens the dialog immediately (`routes.apis.list({ new: true })`).
- With `portal-management` off there is no `Customer portal` row on an API. Do not look for it.
- Name shorter than 3 characters keeps the submit button disabled (`Name must be at least 3 characters long`).
