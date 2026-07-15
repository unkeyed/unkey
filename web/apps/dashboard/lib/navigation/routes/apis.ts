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
import { type WorkspaceScope, buildRoute } from "./shared";

type ApiScope = WorkspaceScope & { apiId: string };
type KeyspaceScope = ApiScope & { keyAuthId: string };
type KeyScope = KeyspaceScope & { keyId: string };

export const apiRoutes = {
  list({ workspaceSlug, new: isNew }: WorkspaceScope & { new?: boolean }): Route {
    return buildRoute("/[workspaceSlug]/apis", { workspaceSlug }, { new: isNew || undefined });
  },

  detail(scope: ApiScope): Route {
    return buildRoute("/[workspaceSlug]/apis/[apiId]", apiParams(scope));
  },

  settings(scope: ApiScope): Route {
    return buildRoute("/[workspaceSlug]/apis/[apiId]/settings", apiParams(scope));
  },

  keys: {
    list(scope: KeyspaceScope): Route {
      return buildRoute("/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]", keyspaceParams(scope));
    },

    detail(scope: KeyScope): Route {
      return buildRoute("/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/[keyId]", keyParams(scope));
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
