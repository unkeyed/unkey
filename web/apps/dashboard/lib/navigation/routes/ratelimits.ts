/**
 * Route builders for the /ratelimits area, exposed as one nested object so call
 * sites read like the url hierarchy: `routes.ratelimits.overrides(scope)`. Every
 * navigable result goes through buildRoute, which checks the bracket pattern
 * against Next's generated route table (typedRoutes) and types the params from
 * the generated ParamMap.
 *
 * Page-level filters (identifiers, since, outcomes, ...) are nuqs state set on the
 * page, never threaded through a navigation call, so they are not builder args.
 */
import type { Route } from "next";
import { type WorkspaceScope, buildRoute } from "./shared";

type NamespaceScope = WorkspaceScope & { namespaceId: string };

export const ratelimitRoutes = {
  list({ workspaceSlug }: WorkspaceScope): Route {
    return buildRoute("/[workspaceSlug]/ratelimits", { workspaceSlug });
  },

  detail(scope: NamespaceScope): Route {
    return buildRoute("/[workspaceSlug]/ratelimits/[namespaceId]", namespaceParams(scope));
  },

  logs(scope: NamespaceScope): Route {
    return buildRoute("/[workspaceSlug]/ratelimits/[namespaceId]/logs", namespaceParams(scope));
  },

  settings(scope: NamespaceScope): Route {
    return buildRoute("/[workspaceSlug]/ratelimits/[namespaceId]/settings", namespaceParams(scope));
  },

  overrides(scope: NamespaceScope): Route {
    return buildRoute(
      "/[workspaceSlug]/ratelimits/[namespaceId]/overrides",
      namespaceParams(scope),
    );
  },
};

function namespaceParams({ workspaceSlug, namespaceId }: NamespaceScope) {
  return { workspaceSlug, namespaceId };
}
