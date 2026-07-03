// Resource URL helpers used by toast actions inside global collections,
// where React Router hooks aren't available. window.location is read at
// click time so the workspace slug always matches whatever route the
// user is currently on.

import type { ScheduledDeletion } from "./scheduled-deletions";

// workspaceSlugFromUrl extracts the leading path segment of the
// current URL — the workspace slug under the (app)/[workspaceSlug]
// layout. Returns null in server contexts or when the path doesn't
// match the workspace layout, in which case the caller should fall
// back to a path that doesn't depend on the slug.
export function workspaceSlugFromUrl(): string | null {
  if (typeof window === "undefined") {
    return null;
  }
  const match = window.location.pathname.match(/^\/([^/]+)/);
  return match?.[1] ?? null;
}

// resourceUrl returns the dashboard route for a given resource type +
// id under the current workspace. Keep in sync with the page layout
// under app/(app)/[workspaceSlug]/.
export function resourceUrl(type: ScheduledDeletion["resourceType"], id: string): string | null {
  const slug = workspaceSlugFromUrl();
  if (!slug) {
    return null;
  }
  switch (type) {
    case "project":
      return `/${slug}/projects/${id}`;
    default:
      return type satisfies never;
  }
}
