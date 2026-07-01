/**
 * Route builders for the /[workspaceSlug]/portal area. `root` is the portal
 * configuration landing page, gated behind the portal-management flag (see
 * app/(app)/[workspaceSlug]/portal/layout.tsx).
 */
import type { Route } from "next";
import { type WorkspaceScope, buildRoute } from "./shared";

export const portalRoutes = {
  root({ workspaceSlug }: WorkspaceScope): Route {
    return buildRoute("/[workspaceSlug]/portal", { workspaceSlug });
  },
};
