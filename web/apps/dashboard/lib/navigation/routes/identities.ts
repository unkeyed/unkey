/**
 * Route builders for the /identities area, exposed as one nested object so call
 * sites read like the url hierarchy: `routes.identities.detail(scope)`. Every
 * navigable result goes through buildRoute, which checks the bracket pattern
 * against Next's generated route table (typedRoutes) and types the params from
 * the generated ParamMap.
 */
import type { Route } from "next";
import { type WorkspaceScope, buildRoute } from "./shared";

type IdentityScope = WorkspaceScope & { identityId: string };

export const identityRoutes = {
  list({ workspaceSlug }: WorkspaceScope): Route {
    return buildRoute("/[workspaceSlug]/identities", { workspaceSlug });
  },

  detail({ workspaceSlug, identityId }: IdentityScope): Route {
    return buildRoute("/[workspaceSlug]/identities/[identityId]", { workspaceSlug, identityId });
  },
};
