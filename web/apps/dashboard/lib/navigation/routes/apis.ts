/**
 * Route builders for the /apis area, exposed as one nested object so call sites
 * read like the url hierarchy: `routes.apis.keys.detail(scope)`. Every navigable
 * result goes through buildRoute, which checks the bracket pattern against Next's
 * generated route table (typedRoutes) and types the params from the generated
 * ParamMap.
 *
 * Page-level filters (keyIds, names, outcomes, since, ...) are nuqs state set on
 * the page, never threaded through a navigation call, so they are not builder
 * query args.
 */
import type { Route } from "next";
import { type ResourceScope, buildRoute } from "./shared";

type ApiScope = ResourceScope & { apiId: string };
type KeyspaceScope = ApiScope & { keyAuthId: string };
type KeyScope = KeyspaceScope & { keyId: string };

const patterns = {
  list: {
    workspace: "/[workspaceSlug]/apis",
    project: "/[workspaceSlug]/projects/[projectId]/keyspaces",
  },
  detail: {
    workspace: "/[workspaceSlug]/apis/[apiId]",
    project: "/[workspaceSlug]/projects/[projectId]/keyspaces/[apiId]",
  },
  portal: {
    workspace: "/[workspaceSlug]/apis/[apiId]/portal",
    project: "/[workspaceSlug]/projects/[projectId]/keyspaces/[apiId]/portal",
  },
  settings: {
    workspace: "/[workspaceSlug]/apis/[apiId]/settings",
    project: "/[workspaceSlug]/projects/[projectId]/keyspaces/[apiId]/settings",
  },
  keys: {
    workspace: "/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]",
    project: "/[workspaceSlug]/projects/[projectId]/keyspaces/[apiId]/keys/[keyAuthId]",
  },
  key: {
    workspace: "/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/[keyId]",
    project: "/[workspaceSlug]/projects/[projectId]/keyspaces/[apiId]/keys/[keyAuthId]/[keyId]",
  },
} as const;

export const apiRoutes = {
  list({ workspaceSlug, projectId, new: isNew }: ResourceScope & { new?: boolean }): Route {
    const query = { new: isNew || undefined };
    return projectId
      ? buildRoute(patterns.list.project, { workspaceSlug, projectId }, query)
      : buildRoute(patterns.list.workspace, { workspaceSlug }, query);
  },

  detail({ projectId, ...scope }: ApiScope): Route {
    return projectId
      ? buildRoute(patterns.detail.project, { ...apiParams(scope), projectId })
      : buildRoute(patterns.detail.workspace, apiParams(scope));
  },

  portal({ projectId, ...scope }: ApiScope): Route {
    return projectId
      ? buildRoute(patterns.portal.project, { ...apiParams(scope), projectId })
      : buildRoute(patterns.portal.workspace, apiParams(scope));
  },

  settings({ projectId, ...scope }: ApiScope): Route {
    return projectId
      ? buildRoute(patterns.settings.project, { ...apiParams(scope), projectId })
      : buildRoute(patterns.settings.workspace, apiParams(scope));
  },

  keys: {
    list({ projectId, ...scope }: KeyspaceScope): Route {
      return projectId
        ? buildRoute(patterns.keys.project, { ...keyspaceParams(scope), projectId })
        : buildRoute(patterns.keys.workspace, keyspaceParams(scope));
    },

    detail({ projectId, ...scope }: KeyScope): Route {
      return projectId
        ? buildRoute(patterns.key.project, { ...keyParams(scope), projectId })
        : buildRoute(patterns.key.workspace, keyParams(scope));
    },
  },
};

function apiParams({ workspaceSlug, apiId }: ApiScope) {
  return { workspaceSlug, apiId };
}

function keyspaceParams({ keyAuthId, ...scope }: KeyspaceScope) {
  return { ...apiParams(scope), keyAuthId };
}

function keyParams({ keyId, ...scope }: KeyScope) {
  return { ...keyspaceParams(scope), keyId };
}
