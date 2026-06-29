/**
 * Route builders for the /logs area, exposed as one nested object so call
 * sites read like the url hierarchy: `routes.logs.list(scope)`. Every
 * navigable result goes through buildRoute, which checks the bracket pattern
 * against Next's generated route table (typedRoutes) and types the params from
 * the generated ParamMap.
 */
import type { Route } from "next";
import { type WorkspaceScope, buildRoute } from "./shared";

export const logRoutes = {
  list({ workspaceSlug }: WorkspaceScope): Route {
    return buildRoute("/[workspaceSlug]/logs", { workspaceSlug });
  },
};
