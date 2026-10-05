# Dashboard feature map

Drive these against a dashboard this skill started. Read `../SKILL.md` first.

## Baseline preconditions

- Doctor verdict is `drivable`, except `auth-gate.md`, which also accepts `auth-landing-only`.
- `AUTH_PROVIDER=local` in `web/apps/dashboard/.env`.
- `FLAGS_LOCAL_OVERRIDES` is empty, so `projects-nav` and `portal-management` stay off.
- Canonical runs have been seeded by `mise run dashboard` (`bin/unkey dev seed local`). Workspace slug `local`, name `Org Local`, API `api_local`, keyspace `ks_local`.
- Base URL `http://127.0.0.1:3000`. One instance. If doctor says `refuse-shared`, do not continue.

## Driving conventions

- Browser accessibility tree. Role plus accessible name from the feature file.
- Sidebar links are anchors whose name is the label in `lib/navigation/leaves.ts` (`Keyspaces (APIs)`, `Ratelimit`, and the rest).
- The workspace crumb link's accessible name is the workspace name (`Org Local` after the default seed). The switch button is `Switch Org Local`.
- Dialogs from `@unkey/ui` expose their `title` as the dialog name. Submit buttons are separate from the opener when the casing differs (`Create keyspace` vs `Create Keyspace`).
- Do not type a real customer secret into the UI. A key created here is local seed data. Copy it only into the evidence file for this run if the success dialog shows it, and do not commit the secret itself. Record that the heading `Key Created` appeared.

## Proof and skip reporting

For each feature you touch, append a short note under the run's evidence directory:

- `proved`: URL, control, resulting heading or URL, screenshot or HTML path
- `skipped`: the doctor verdict or missing local dependency that made the path unsafe

Do not mark a feature proved from a Vitest file. Name the test file as non-UI evidence when the browser path is blocked.

## Feature entry contract

Each feature file has an H1 and a short description, then these H2 sections in order:

1. Sub-features
2. How to get to it (user POV)
3. Driving it with the browser
4. Gotchas
