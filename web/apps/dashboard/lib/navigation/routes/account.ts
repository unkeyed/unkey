/**
 * Route builder for the account area. Account settings belong to the user, not
 * the workspace, but the page renders inside the workspace shell, so the path
 * still carries a workspace slug.
 */
import type { Route } from "next";
import { type WorkspaceScope, buildRoute } from "./shared";

export const accountRoutes = {
  overview({ workspaceSlug }: WorkspaceScope): Route {
    return buildRoute("/[workspaceSlug]/account", { workspaceSlug });
  },
};
