import type { Route } from "next";
import { type WorkspaceScope, buildRoute } from "./shared";

export const metricsRoutes = {
  list({ workspaceSlug }: WorkspaceScope): Route {
    return buildRoute("/[workspaceSlug]/metrics", { workspaceSlug });
  },
};
