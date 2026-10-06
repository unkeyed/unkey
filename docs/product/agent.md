---
title: Sign up a user
sidebarTitle: Agent signup
description: "Register through WorkOS Agent Registration, then create a root key for a new or existing workspace."
sources:
  - web/apps/dashboard/app/api/agent/workspace/route.ts
  - web/apps/dashboard/app/api/agent/root-key/route.ts
  - web/apps/dashboard/app/api/agent/auth-md/route.ts
  - web/apps/dashboard/app/.well-known/oauth-protected-resource/route.ts
  - web/apps/dashboard/lib/agent-signup/verify.ts
  - web/apps/dashboard/lib/workspace/create-workspace.ts
  - web/apps/dashboard/lib/auth/workos.ts
  - web/apps/dashboard/lib/trpc/routers/workspace/create.ts
  - web/apps/dashboard/proxy.ts
  - https://workos.com/docs/authkit/agent-registration
  - https://workos.com/docs/authkit/metadata
  - https://workos.com/docs/reference/organization/create
  - https://workos.com/docs/reference/authkit/agent-registration/register
  - https://workos.com/docs/reference/authkit/agent-registration/claim-complete
  - https://workos.com/docs/reference/agents/registration/get-registration
  - https://workos.com/auth-md/docs/auth-md
  - https://workos.com/auth-md/docs/apps
---

Unkey issues API keys and runs apps. Follow this page to register an agent for a person, including a person who already has an Unkey account.

Registration happens on the WorkOS AuthKit domain. Unkey doesn't implement `POST /agent/identity`. After the person claims the registration, you call Unkey with the access token WorkOS returns.

## Follow these rules

Use `service_auth` only. Don't register as `anonymous`. Don't send an `identity_assertion`.

Don't bypass the claim page. The person signs in on AuthKit and reads you the `user_code`. If they already belong to a WorkOS organization, AuthKit asks them to pick one. Finish that selection, complete the claim, and continue on this page. Don't stop and send them to dashboard sign-in instead.

Hosted AuthKit runs a Cloudflare Turnstile check that WorkOS operates. If Radar is enabled for that WorkOS environment, it can also block or challenge a bot, including an AI agent. This repository does not implement Turnstile or Radar.

Don't create a WorkOS user with the email marked verified, and don't read a Magic Auth code from an API response. That skips the email check.

Store the root key when Unkey returns it. Unkey shows that secret once.

## Discover the service

Fetch `https://app.unkey.com/auth.md`. That URL reverse-proxies the WorkOS-generated skill at `https://<authkit-domain>/agent/auth.md`, with a short Unkey preface that points at the workspace and root-key calls below. WorkOS documents the hosted skill URL as public, so you can also open it directly. Use this page for the Unkey calls. The generated skill doesn't describe them.

A 401 from the Unkey agent endpoints includes:

```http
WWW-Authenticate: Bearer resource_metadata="https://app.unkey.com/.well-known/oauth-protected-resource"
```

`GET https://app.unkey.com/.well-known/oauth-protected-resource` returns the protected resource document. Its `authorization_servers` value is the AuthKit origin. Fetch that origin's `/.well-known/oauth-authorization-server` and read `agent_auth` for `identity_endpoint`, `claim_endpoint`, and `skill`.

If `auth.md` or the protected resource document returns 404, this Unkey environment doesn't have agent signup configured. Stop. Don't invent another signup URL.

## Register

`POST https://<authkit-domain>/agent/identity` with `Content-Type: application/json`:

```json
{"type": "service_auth", "login_hint": "person@example.com"}
```

Use the person's email as `login_hint`. The response includes `claim.token` and `claim.attempt.verification_uri`. Keep `claim.token`.

This call is documented at [Register an agent](https://workos.com/docs/reference/authkit/agent-registration/register) and [Agent Registration](https://workos.com/docs/authkit/agent-registration).

## The person claims the agent

Give the person `verification_uri`. That page is hosted by AuthKit. Unkey doesn't host a claim form.

Tell them: "Open this AuthKit page and sign in with your own email. If it asks you to choose an organization, pick one you already belong to. If it shows a bot check, complete it. Then read me the code on the page."

The API reference shows a `user_code` shaped like `BCDF-GHJK`. The auth.md protocol text also describes a 6-digit code. Send the code exactly as the page shows it. Don't reformat it.

Then `POST https://<authkit-domain>/agent/identity/claim/complete`:

```json
{"claim_token": "<claim.token>", "user_code": "<code from the person>"}
```

On success the body contains `identity.assertion`, `identity.expires_at`, and `identity.refresh_token` (`value` and `expires_at`). WorkOS delivers that verified identity in this response once, so save the assertion and `identity.refresh_token.value` immediately. You exchange the assertion for an access token below, and you use the refresh token when that assertion expires.

If the first attempt expires, `POST https://<authkit-domain>/agent/identity/claim` with `type`, `claim_token`, and `login_hint`, then use the new `verification_uri`. Don't call that endpoint while the original attempt is still valid.

Documented errors include `invalid_user_code`, `user_code_expired`, `claim_not_confirmed`, `claim_expired`, `claim_revoked`, and `already_claimed`. For `claim_not_confirmed`, wait and retry the complete call. For an expired or revoked claim, start registration again.

## Exchange the assertion

`POST https://<authkit-domain>/oauth2/token` with `Content-Type: application/x-www-form-urlencoded`:

```bash
curl -X POST "https://<authkit-domain>/oauth2/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=urn:ietf:params:oauth:grant-type:jwt-bearer" \
  --data-urlencode "assertion=<identity.assertion>"
```

Use the `access_token` from the response. Unkey accepts that access token only. An agent API key is not a JWT, and Unkey rejects it. Don't send the identity assertion to Unkey.

The [auth.md protocol](https://workos.com/auth-md/docs/auth-md) describes this response as `access_token`, `token_type`, `expires_in`, and `scope`, with no `refresh_token`. The [Agent Registration guide](https://workos.com/docs/authkit/agent-registration) does not publish a JSON example for this call. The access token lifetime is the credential expiration configured for the WorkOS environment. WorkOS does not publish a fixed number of seconds.

`aud` must be the audience configured for this Unkey environment, or the `resource` URL from `/.well-known/oauth-protected-resource`. Unkey accepts those two values and no others. The documented fields are `grant_type` and `assertion`. This grant does not take `client_id`. `resource` is optional. Add `--data-urlencode "resource=https://app.unkey.com"` when you want `aud` to be the protected resource URL.

The same assertion can be exchanged again until it expires. Claim complete is the one response that delivers the identity.

## Replace an expired access token

Get a fresh access token before each Unkey call. If Unkey returns 401 `expired`, do the same thing and retry that Unkey call once.

Exchange the saved assertion with the call above. Read `expires_in` when the response includes it, and exchange again before that access token expires.

When `identity.expires_at` has passed, or `/oauth2/token` returns `invalid_grant`, rotate the assertion with the refresh token from claim complete:

```bash
curl -X POST "https://<authkit-domain>/agent/identity" \
  -H "Content-Type: application/json" \
  -d '{"type":"refresh","refresh_token":"<identity.refresh_token.value>"}'
```

That body is JSON. The field is `refresh_token`. There is no `grant_type`, `client_id`, or `resource` on this call. WorkOS returns a fresh identity assertion and a rotated refresh token. It does not publish a separate JSON example for this response, and it does not say the previous refresh token stops working. Save `identity.assertion` and `identity.refresh_token.value` from the response when they are present, then exchange that assertion at `/oauth2/token`.

`invalid_refresh_token` means the refresh token is expired, revoked, or invalid. If the refresh token is missing, past `identity.refresh_token.expires_at`, or rejected, start again with `POST /agent/identity` and a new claim. Don't reuse the old claim token.

## Choose a workspace

The same access token works for a new account and for an existing one. Unkey treats the token's `act.sub` claim as the WorkOS user who authorized the agent. Unkey mints a root key only when that user is an active admin of the workspace.

If the person already admins a workspace, skip workspace creation. Call `POST /api/agent/root-key` with that workspace's `workspaceId` or `slug`.

If they need a workspace, call `POST /api/agent/workspace` with a new `name` and `slug`. Use that call when they have no workspace, and use it again when they want another one. Then create a root key for the workspace you just created.

## Create the workspace

`POST https://app.unkey.com/api/agent/workspace`:

```bash
curl -X POST https://app.unkey.com/api/agent/workspace \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Acme","slug":"acme"}'
```

`name` is 3 to 50 characters. `slug` is 3 to 64 characters: lowercase letters, numbers, and single hyphens, with no leading or trailing hyphen. The slug is the workspace URL handle and cannot be changed later. It must be unique across Unkey.

A 200 response is:

```json
{"workspaceId":"ws_...","orgId":"org_...","slug":"acme","name":"Acme"}
```

Unkey creates a WorkOS organization, a free-tier workspace, and an admin membership for the user who authorized the claim (`act.sub`). That is the same creation path as the signed-in dashboard mutation `workspace.create`. Call it again with another slug to create another workspace. The same registration can create more than one. A taken slug returns 409 `slug_taken`. Don't send this call when the person already admins the workspace you want a root key for.

Unkey stores this registration's id on the new WorkOS organization as metadata `agent_registration_id`. Organization metadata is not unique. See [Metadata and External IDs](https://workos.com/docs/authkit/metadata) and [Create an organization](https://workos.com/docs/reference/organization/create).

A signed-in person can still create a workspace in the dashboard at `https://app.unkey.com/new`. You don't need that page for this flow.

## Create a root key

`POST https://app.unkey.com/api/agent/root-key` with a current access token. Exchange the assertion again if the token you used for workspace creation has expired. Send `workspaceId` from the workspace response, or `slug`. You must be an active admin of that workspace. Unkey loads the workspace, then checks your WorkOS organization membership and role. The body does not grant access by itself.

```bash
curl -X POST https://app.unkey.com/api/agent/root-key \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"workspaceId":"ws_...","name":"agent"}'
```

A missing workspace, or a `workspaceId` and `slug` that do not refer to the same workspace, returns 404 `workspace_not_found`. A member who is not an admin, and a user who is not a member, returns 403 `forbidden`. You can create more than one root key. Each key is scoped to that workspace.

Omit `permissions` to receive the default set: read and write on `projects/*/keyspaces/*`, plus read, write, and verify on `projects/*/keyspaces/*/keys/*`. That set can create and verify API keys. It cannot decrypt key material, delete resources, or manage root keys.

You can request a smaller set, or add delete and keyspace log read, using `path` and `action`:

```json
{
  "workspaceId": "ws_...",
  "name": "agent",
  "permissions": [
    {"path": "projects/*/keyspaces/*", "action": "read"},
    {"path": "projects/*/keyspaces/*/keys/*", "action": "verify"}
  ]
}
```

Allowed paths are `projects/*/keyspaces/*`, `projects/*/keyspaces/*/logs`, and `projects/*/keyspaces/*/keys/*`. Keyspaces accept read, write, and delete. Logs accept read. Keys accept read, write, delete, and verify. `verify` applies only to keys. `decrypt`, `rootKeys/*`, and `**` are rejected even though a workspace admin could grant them in the dashboard.

A 200 response is:

```json
{"keyId":"key_...","key":"unkey_...","permissions":["unkey:v1:ws_...:projects/*/keyspaces/*#read"]}
```

Copy `key` before you discard the response. Unkey stores the key hash, plus its id, name, prefix, start, and end. It does not store the plaintext secret.

The root key is created through `POST /v2/rootKeys.createAgentKey`, which also writes the root-key audit events. That route accepts only the permissions listed above. You can't call it yourself.

`unkey auth login` prompts `Enter your root key:` and stores whatever you type in `~/.unkey/config.toml`. Run it only after you have this secret. See [unkey auth login](/platform/cli/auth/login).

## Failures

| Status | `error` | What to do |
| --- | --- | --- |
| 401 | `expired` | Exchange the saved identity assertion for a new access token. If that assertion is expired or the exchange returns `invalid_grant`, rotate it with `POST /agent/identity` and `type: refresh`, then exchange again. Retry this Unkey call once. If the refresh token is missing or rejected, register and claim again. |
| 401 | `missing_token`, `invalid_token`, `wrong_audience` | Send a current access token. `aud` must be this environment's configured audience or the `resource` value from the protected resource document. |
| 403 | `unclaimed`, `anonymous`, `unsupported_registration` | Finish a `service_auth` claim for this user. Don't retry with an anonymous registration. |
| 403 | `user_mismatch` | The access token's authorizing user does not match the user who completed the claim. Start registration again and claim it as that person. |
| 403 | `forbidden` | Sign in as an admin of the workspace you named. |
| 409 | `slug_taken` | Pick another slug. |
| 429 | `rate_limited` | Wait and retry. |
| 404 | `workspace_not_found` | Check `workspaceId` and `slug`. They must name one existing workspace. |
| 404 | `not_found` | This environment has agent signup turned off. |

## Continue after the root key

Create a keyspace, then a key. Dashboard and API both work once the root key exists.

1. Open `https://app.unkey.com/<slug>/apis?new=true`. The **Create keyspace** dialog asks for **Name** (required, 3 to 50 characters on this form). Click **Create Keyspace**. The page redirects to that keyspace.
2. On the keyspace page, click **Create key**. **Name** is optional and at most 256 characters. The **Create key** button stays disabled until the form is valid. Copy the secret when it is shown. It is shown once.
3. Or call the API with the root key. `POST https://api.unkey.com/v2/apis.createApi` takes `{ "name": "..." }` (`name` is 3 to 256 characters). `POST https://api.unkey.com/v2/keys.createKey` takes `{ "apiId": "api_...", "name": "My first key" }`. Send `Authorization: Bearer <root key>`.

Read these next:

- [Issue and verify your first key](/api-management/get-started/quickstart)
- [Configure root keys](/platform/root-keys/configure)
- [API authentication](/platform/api/authentication)
- [Workspaces](/platform/workspace/overview)
- [Install the CLI](/platform/cli/install)

A Compute deploy needs a plan. Sign-up does not. The person picks one later under **Settings**, then **Billing**. See [Plans](/platform/billing/plans).
