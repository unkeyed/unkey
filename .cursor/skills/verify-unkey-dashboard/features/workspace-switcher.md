# Workspace switcher

The top bar names the current workspace and opens a list of orgs the session can enter.

## Sub-features

- Crumb link whose accessible name is the workspace name (`Org Local` after the default seed). It links to the workspace root, which replaces to the projects list.
- Button `Switch {workspace name}` (`Switch Org Local`).
- Search box placeholder `Find workspace...`.
- Empty copy `No workspaces found` and the footer link `New workspace` to `/new`.
- Load failure is an alert `Unable to load workspaces` with button `Try again`.
- Loading copy `Loading workspaces...`.

## How to get to it (user POV)

From any signed-in page (projects, keyspaces, settings), the top bar starts with the Unkey mark (`aria-label` `Unkey`), then the workspace name. The chevron button beside the name is the switcher. Account actions are a separate button, `Account menu`, with `Account settings` and `Sign out`.

## Driving it with the browser

Requires verdict `drivable`.

1. Open `http://127.0.0.1:3000/local/projects` and confirm the heading `Projects`.
2. Confirm the link named `Org Local`.
3. Activate the button named `Switch Org Local`.
4. Confirm the textbox placeholder `Find workspace...` and a row for the current org. Local auth has a single organization, `org_localdefault`, so the list has one workspace.
5. Activate `New workspace` only if you intend to prove onboarding. It navigates to `/new`. Otherwise close the popover and stop. The proved state is the open list containing `Org Local`.

Do not call `switchToOrg` with another org id. `LocalAuthProvider.switchOrg` throws `Organization {id} not found` unless the id is `org_localdefault`.

## Gotchas

- The switch button's accessible name includes the live workspace name. After a rename in Settings (`General`), the name changes. The default seed name is `Org Local`.
- `listAvailable` errors render `Unable to load workspaces`, not an empty product state. `No workspaces found` is the empty list.
- Local mode cannot demonstrate a two-workspace switch. Proving the popover is the whole local path.
