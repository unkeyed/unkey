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
import { type ResourceScope, scopedRoute } from "./shared";

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
  list({ new: isNew, ...scope }: ResourceScope & { new?: boolean }): Route {
    return scopedRoute(patterns.list, scope, { new: isNew || undefined });
  },

  detail(scope: ApiScope): Route {
    return scopedRoute(patterns.detail, scope);
  },

  portal(scope: ApiScope): Route {
    return scopedRoute(patterns.portal, scope);
  },

  settings(scope: ApiScope): Route {
    return scopedRoute(patterns.settings, scope);
  },

  keys: {
    list(scope: KeyspaceScope): Route {
      return scopedRoute(patterns.keys, scope);
    },

    detail(scope: KeyScope): Route {
      return scopedRoute(patterns.key, scope);
    },
  },
};
