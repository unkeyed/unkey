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
import { type ResourceScope, buildRoute } from "./shared";

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
  list({ workspaceSlug, projectId }: ResourceScope): Route {
    return projectId
      ? buildRoute(patterns.list.project, { workspaceSlug, projectId })
      : buildRoute(patterns.list.workspace, { workspaceSlug });
  },

  detail({ projectId, ...scope }: NamespaceScope): Route {
    return projectId
      ? buildRoute(patterns.detail.project, { ...namespaceParams(scope), projectId })
      : buildRoute(patterns.detail.workspace, namespaceParams(scope));
  },

  logs({ projectId, ...scope }: NamespaceScope): Route {
    return projectId
      ? buildRoute(patterns.logs.project, { ...namespaceParams(scope), projectId })
      : buildRoute(patterns.logs.workspace, namespaceParams(scope));
  },

  settings({ projectId, ...scope }: NamespaceScope): Route {
    return projectId
      ? buildRoute(patterns.settings.project, { ...namespaceParams(scope), projectId })
      : buildRoute(patterns.settings.workspace, namespaceParams(scope));
  },

  overrides({ projectId, ...scope }: NamespaceScope): Route {
    return projectId
      ? buildRoute(patterns.overrides.project, { ...namespaceParams(scope), projectId })
      : buildRoute(patterns.overrides.workspace, namespaceParams(scope));
  },
};

function namespaceParams({ workspaceSlug, namespaceId }: NamespaceScope) {
  return { workspaceSlug, namespaceId };
}
