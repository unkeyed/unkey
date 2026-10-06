---
title: Sign up a new user
sidebarTitle: Agent signup
description: "Steps for an agent to sign a person up for Unkey and create their first workspace."
sources:
  - web/apps/dashboard/proxy.ts
  - web/apps/dashboard/app/auth/sign-up/[[...sign-up]]/page.tsx
  - web/apps/dashboard/app/auth/sso-callback/[[...sso-callback]]/route.ts
  - web/apps/dashboard/app/api/webhooks/workos/route.ts
  - web/apps/dashboard/app/(app)/layout.tsx
  - web/apps/dashboard/app/new/hooks/use-workspace-step.tsx
  - web/apps/dashboard/lib/trpc/routers/workspace/create.ts
  - web/apps/dashboard/app/(app)/[workspaceSlug]/root-keys/page.tsx
  - web/apps/dashboard/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/schema.ts
  - web/apps/dashboard/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/lib/templates.ts
  - cmd/auth/main.go
---

Unkey issues API keys and runs apps. Follow this page when you are creating an account for a person who does not have one yet.

## Follow these rules

Create one account. If the person already has an Unkey account, open the sign-in URL below and stop. Do not create a second account.

Do not bypass a bot check, an email code, or an OAuth consent screen. Hosted AuthKit runs a Cloudflare Turnstile check that WorkOS operates. The person completes it. If Radar is enabled for that WorkOS environment, it can also block or challenge a bot, including an AI agent. This repository does not implement Turnstile or Radar.

Ask the person for a verification code only so you can type it into the WorkOS page that sent it. Do not send that code anywhere else.

Do not call an API to sign up or create a workspace. Unkey has no such endpoint. Do not call `POST /v2/rootKeys.createKey` for the first key. That route requires a root key that already exists.

WorkOS Agent Auth, Agent Registration, CLI device authorization, Connect, and the user-management APIs do not replace the hosted page. Agent Auth and machine-to-machine tokens act inside an organization that already exists. Agent Registration and CLI device authorization still send the person through sign-in. Creating a WorkOS user with the email marked verified, or reading a Magic Auth code from the API response, skips the email check. Do not do that. None of these calls create an Unkey workspace. `workspace.create` runs only for a signed-in dashboard session.

## Sign the person up

Production sign-up does not render a form in this repository. `GET https://app.unkey.com/auth/sign-up` redirects to the WorkOS AuthKit hosted page with a sign-up screen hint. The callback is `https://app.unkey.com/auth/sso-callback`, which then sends the browser to `/apis`.

1. Open `https://app.unkey.com/auth/sign-up`.
2. Stop when the browser leaves `app.unkey.com` for the WorkOS page. Tell the person: "Unkey sign-up is on this WorkOS page. Use your own email. If it sends a code, read the code to me or type it on that page. If it asks you to approve an OAuth provider, finish that approval. If it shows a bot check, complete it. I will not bypass it."
3. Wait until the browser is back on `app.unkey.com`. The callback opens `/apis`. When the account has no workspace, the app shell then opens `https://app.unkey.com/new`.

Email-code sign-up stays unverified until that code is confirmed. OAuth sign-up arrives already verified. You cannot see which methods the hosted page offers, because they are not configured in this repository. The person has to finish that page.

If they already have an account, open `https://app.unkey.com/auth/sign-in` instead and give them the same handoff.

If the browser lands on `https://app.unkey.com/auth/error`, the attempt failed. The page says "We could not sign you in" and links to sign-in. Start again at `https://app.unkey.com/auth/sign-up`. If it keeps failing, the person emails [support@unkey.com](mailto:support@unkey.com).

Ignore the local dashboard. With `AUTH_PROVIDER=local`, `/auth/sign-up` does not create an account.

## Create the first workspace

Open `https://app.unkey.com/new` if the person is signed in and has no workspace. The app shell also sends them there when the session has no organization.

The page title is **Create Company Workspace**. The description is "Name your workspace and choose its URL."

1. Fill **Workspace name**. It is required, 3 to 50 characters after trimming.
2. Fill **Workspace URL handle**. It is required, 3 to 64 characters: lowercase letters, numbers, and single hyphens, with no leading or trailing hyphen. The field is prefixed with `app.unkey.com/`. Once the name is at least 3 characters, the form fills the handle from the name. Edit it before submitting if you want a different handle. The handle is the slug in every dashboard URL and cannot be changed later. It must be unique across Unkey. If it is taken, the form reports "A workspace with this slug already exists." Pick another handle.
3. Click **Create workspace**.

That submits the signed-in dashboard mutation `workspace.create` with `name` and `slug`. It is not a public HTTP endpoint. The workspace starts on the free API tier. No card is required. The creator becomes the workspace admin.

The dashboard then switches into that workspace. With the default navigation it opens `https://app.unkey.com/<slug>/apis`. When the projects navigation flag is on, it opens `https://app.unkey.com/<slug>/projects`. Read `<slug>` from the address bar.

## Create a root key

Open `https://app.unkey.com/<slug>/root-keys`. You must still be the workspace admin. `https://app.unkey.com/<slug>/settings/root-keys` is the same page, and redirects to the first URL when projects navigation is on.

1. Click **New Root Key**.
2. Fill **Name**. It is required. Use a name that says what the key is for, such as `agent`.
3. Under **Permissions**, pick one template. **All write permissions** can create keyspaces and keys. **All read permissions**, **Verify keys**, and **Standalone ratelimiting** are narrower. **Start new** begins an empty policy. At least one permission is required.
4. Click **Create key**.
5. The dialog title is **Root Key created**. Copy the secret before you click **Done**. It is shown only once and starts with `unkey_`.

`POST https://api.unkey.com/v2/rootKeys.createKey` cannot create this first key. That call requires an existing root key with permission to write root keys. The dashboard creates the first key with the signed-in session.

On `main`, `unkey auth login` does not sign anyone in. It prompts `Enter your root key:` and stores whatever you type in `~/.unkey/config.toml`. Run it only after the dashboard has shown you a secret. See [unkey auth login](/platform/cli/auth/login).

`unkey login` device approval is not on `main`. It is on branch `eng-2994-add-device-code-login-flow-to-unkey-cli`, which adds `unkey login`, a `https://app.unkey.com/cli/device` approval page, and `POST /v2/cli.startDeviceLogin`. Do not run that command until the branch is on `main`. After it lands, the person still has to approve the code in the dashboard while signed in as a workspace admin.

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
