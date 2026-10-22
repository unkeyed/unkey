import type { Route } from "next";
import { type WorkspaceScope, buildRoute } from "./shared";

export const envVarRoutes = {
  list({ workspaceSlug }: WorkspaceScope): Route {
    return buildRoute("/[workspaceSlug]/env-vars", { workspaceSlug });
  },
};
