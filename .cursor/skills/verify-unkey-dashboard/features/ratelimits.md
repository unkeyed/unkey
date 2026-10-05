# Ratelimits

Namespace list for workspace ratelimits. The sidebar label is `Ratelimit`. The page heading is `Ratelimits`.

## Sub-features

- List heading `Ratelimits` at `/{workspaceSlug}/ratelimits`.
- Button `Create namespace`. Dialog title `Create Namespace`. Name field label `Name`, placeholder `email.outbound`.
- Submit button `Create Namespace`, or `Creating...` while the mutation runs.
- Success toast `Namespace created`.
- Inside a namespace, section links `Requests`, `Logs`, `Settings`, `Overrides`.

## How to get to it (user POV)

From a drivable instance, activate the sidebar link `Ratelimit`. The URL is `/local/ratelimits`. The header reads `Ratelimits`.

With `projects-nav` on, this list moves under a project (`/local/projects/{projectId}/ratelimits`) and the workspace sidebar says `Ratelimits` only inside the project. Leave the flag off for this file.

## Driving it with the browser

Requires verdict `drivable`.

1. Activate the link named `Ratelimit`.
2. Confirm the heading `Ratelimits`.
3. Activate the button named `Create namespace`.
4. Fill the textbox named `Name` with `verify.ratelimit` (letters, digits, `_`, `-`, and `.` only).
5. Activate `Create Namespace`.
6. Confirm the toast `Namespace created` and that the dialog closes. The new name in the list is the side effect.

Opening a namespace shows the heading area for that namespace and the links `Requests`, `Logs`, `Settings`, and `Overrides`. Requests is `/local/ratelimits/{namespaceId}`.

## Gotchas

- Sidebar accessible name is `Ratelimit`. Page heading is `Ratelimits`. Dialog title and submit use `Create Namespace` with a capital N. The opener is `Create namespace`.
- An existing name sets the field error `Namespace already exists`.
- The mutation is `trpc.ratelimit.namespace.create`. It needs MySQL. It does not need a browser harness beyond the page.
