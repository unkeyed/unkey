# Auth gate

Local mode replaces WorkOS AuthKit with a single screen that continues as the built-in admin. The sign-in route is `/auth/sign-in`.

## Sub-features

- Local landing with heading `Local dashboard` and the sentence `Authentication is disabled in local mode. Continue with the built-in local account.`
- Link `Continue to dashboard` (`href="/apis"`).
- WorkOS mode does not render this landing. The proxy sends `/auth/sign-in` to AuthKit. If the page module runs anyway it redirects to `/auth/error?reason=entry`, whose heading is `We could not sign you in` and whose link is `Sign in again`. This skill does not drive that mode.

## How to get to it (user POV)

Open `http://127.0.0.1:3000/auth/sign-in`. The auth layout also shows a `Documentation` link to `https://www.unkey.com/docs`. The landing itself is the heading and the continue link from `app/auth/local-auth-landing.tsx`.

No password field is rendered. `updateLocalSession` mints the `unkey-session` cookie with token `local_session_token` when a server component is allowed to set cookies.

## Driving it with the browser

Allowed when doctor prints `drivable` or `auth-landing-only`.

1. Open `http://127.0.0.1:3000/auth/sign-in`.
2. Find the heading named `Local dashboard`.
3. Find the link named `Continue to dashboard` and confirm its href is `/apis`.
4. Stop when the verdict is `auth-landing-only`. Activating the link requests a workspace from MySQL, which is not running.
5. When the verdict is `drivable`, activate `Continue to dashboard`. The address bar leaves `/auth/sign-in`. The next document is the projects list for the session workspace (heading `Projects`) after the client replace in `app/(app)/[workspaceSlug]/page.tsx`. If the database has no workspace for `org_localdefault`, the shell sends the browser to `/new` (heading text `Create workspace` on the onboarding step).

Non-UI evidence when the server cannot start:

```bash
mise exec -- pnpm --dir=web/apps/dashboard exec vitest run lib/auth/tests/local.test.ts lib/navigation/routes/auth.test.ts
```

Those tests check `LocalAuthProvider` and `routes.auth.signIn() === "/auth/sign-in"`. They do not open a browser.

## Gotchas

- `Continue to dashboard` does not open the keyspace list. `/apis` is parsed as a workspace slug. Use the sidebar `Keyspaces (APIs)` or `/local/apis` for keyspaces.
- Do not follow the `Documentation` link. It leaves the local app.
- An `AUTH_PROVIDER=workos` document is not a local proof. Doctor exits 4 before launch.
- Next may log that HMR from `127.0.0.1` is blocked. The sign-in document at `http://127.0.0.1:3000/auth/sign-in` still returns the heading. Do not switch the drive to a public host to quiet that log.
