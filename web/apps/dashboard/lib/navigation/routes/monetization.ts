/**
 * Route builders for the /monetization area (end-user billing: a customer
 * billing their own users). Exposed as one nested object so call sites read
 * like the url hierarchy: `routes.monetization.overview(scope)`.
 */
import type { Route } from "next";
import { type WorkspaceScope, buildRoute } from "./shared";

export const monetizationRoutes = {
  overview({ workspaceSlug }: WorkspaceScope): Route {
    return buildRoute("/[workspaceSlug]/monetization", { workspaceSlug });
  },
};
