/**
 * Route builders for the /audit area, exposed as one nested object so call
 * sites read like the url hierarchy: `routes.audit.list(scope)`. Every
 * navigable result goes through buildRoute, which checks the bracket pattern
 * against Next's generated route table (typedRoutes) and types the params from
 * the generated ParamMap.
 */
import type { Route } from "next";
import { type WorkspaceScope, buildRoute } from "./shared";

export const auditRoutes = {
  list({ workspaceSlug }: WorkspaceScope): Route {
    return buildRoute("/[workspaceSlug]/audit", { workspaceSlug });
  },
};
