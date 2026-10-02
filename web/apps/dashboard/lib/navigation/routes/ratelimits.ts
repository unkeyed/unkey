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
import { type ResourceScope, scopedRoute } from "./shared";

type NamespaceScope = ResourceScope & { namespaceId: string };

const patterns = {
  list: {
    workspace: "/[workspaceSlug]/ratelimits",
    project: "/[workspaceSlug]/projects/[projectId]/ratelimits",
  },
  detail: {
    workspace: "/[workspaceSlug]/ratelimits/[namespaceId]",
    project: "/[workspaceSlug]/projects/[projectId]/ratelimits/[namespaceId]",
  },
  logs: {
    workspace: "/[workspaceSlug]/ratelimits/[namespaceId]/logs",
    project: "/[workspaceSlug]/projects/[projectId]/ratelimits/[namespaceId]/logs",
  },
  settings: {
    workspace: "/[workspaceSlug]/ratelimits/[namespaceId]/settings",
    project: "/[workspaceSlug]/projects/[projectId]/ratelimits/[namespaceId]/settings",
  },
  overrides: {
    workspace: "/[workspaceSlug]/ratelimits/[namespaceId]/overrides",
    project: "/[workspaceSlug]/projects/[projectId]/ratelimits/[namespaceId]/overrides",
  },
} as const;

export const ratelimitRoutes = {
  list(scope: ResourceScope): Route {
    return scopedRoute(patterns.list, scope);
  },

  detail(scope: NamespaceScope): Route {
    return scopedRoute(patterns.detail, scope);
  },

  logs(scope: NamespaceScope): Route {
    return scopedRoute(patterns.logs, scope);
  },

  settings(scope: NamespaceScope): Route {
    return scopedRoute(patterns.settings, scope);
  },

  overrides(scope: NamespaceScope): Route {
    return scopedRoute(patterns.overrides, scope);
  },
};
