/**
 * Route builders for the /authorization area, exposed as one nested object so
 * call sites read like the url hierarchy: `routes.authorization.roles(scope)`.
 * Every navigable result goes through buildRoute, which checks the bracket
 * pattern against Next's generated route table (typedRoutes) and types the
 * params from the generated ParamMap.
 */
import type { Route } from "next";
import { type WorkspaceScope, buildRoute } from "./shared";

export const authorizationRoutes = {
  roles({ workspaceSlug }: WorkspaceScope): Route {
    return buildRoute("/[workspaceSlug]/authorization/roles", { workspaceSlug });
  },

  permissions({ workspaceSlug }: WorkspaceScope): Route {
    return buildRoute("/[workspaceSlug]/authorization/permissions", { workspaceSlug });
  },
};
