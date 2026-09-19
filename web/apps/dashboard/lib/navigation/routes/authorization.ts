/**
 * Route builders for the /authorization area, exposed as one nested object so
 * call sites read like the url hierarchy: `routes.authorization.roles(scope)`.
 * Every navigable result goes through buildRoute, which checks the bracket
 * pattern against Next's generated route table (typedRoutes) and types the
 * params from the generated ParamMap.
 */
import type { Route } from "next";
import { type ResourceScope, scopedRoute } from "./shared";

const patterns = {
  roles: {
    workspace: "/[workspaceSlug]/authorization/roles",
    project: "/[workspaceSlug]/projects/[projectId]/authorization/roles",
  },
  permissions: {
    workspace: "/[workspaceSlug]/authorization/permissions",
    project: "/[workspaceSlug]/projects/[projectId]/authorization/permissions",
  },
} as const;

export const authorizationRoutes = {
  roles(scope: ResourceScope): Route {
    return scopedRoute(patterns.roles, scope);
  },

  permissions(scope: ResourceScope): Route {
    return scopedRoute(patterns.permissions, scope);
  },
};
