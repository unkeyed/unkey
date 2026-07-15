/**
 * Route builders for the top-level /[workspaceSlug]/root-keys area. Root Keys
 * moved out of settings into the workspace sidebar; the old
 * /settings/root-keys URL redirects here.
 */
import type { Route } from "next";
import { type WorkspaceScope, buildRoute } from "./shared";

export const rootKeyRoutes = {
  list({ workspaceSlug }: WorkspaceScope): Route {
    return buildRoute("/[workspaceSlug]/root-keys", { workspaceSlug });
  },
};
