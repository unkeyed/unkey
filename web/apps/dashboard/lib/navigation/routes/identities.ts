/**
 * Route builders for the /identities area, exposed as one nested object so call
 * sites read like the url hierarchy: `routes.identities.detail(scope)`. Every
 * navigable result goes through buildRoute, which checks the bracket pattern
 * against Next's generated route table (typedRoutes) and types the params from
 * the generated ParamMap.
 */
import type { Route } from "next";
import { type ResourceScope, buildRoute } from "./shared";

type IdentityScope = ResourceScope & { identityId: string };

const patterns = {
  list: {
    workspace: "/[workspaceSlug]/identities",
    project: "/[workspaceSlug]/projects/[projectId]/identities",
  },
  detail: {
    workspace: "/[workspaceSlug]/identities/[identityId]",
    project: "/[workspaceSlug]/projects/[projectId]/identities/[identityId]",
  },
} as const;

export const identityRoutes = {
  list({ workspaceSlug, projectId }: ResourceScope): Route {
    return projectId
      ? buildRoute(patterns.list.project, { workspaceSlug, projectId })
      : buildRoute(patterns.list.workspace, { workspaceSlug });
  },

  detail({ workspaceSlug, projectId, identityId }: IdentityScope): Route {
    return projectId
      ? buildRoute(patterns.detail.project, { workspaceSlug, projectId, identityId })
      : buildRoute(patterns.detail.workspace, { workspaceSlug, identityId });
  },
};
