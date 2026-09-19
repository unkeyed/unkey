import type { Route } from "next";
import { type WorkspaceScope, buildRoute } from "./shared";

export const rootKeyRoutes = {
  list({ workspaceSlug }: WorkspaceScope): Route {
    return buildRoute("/[workspaceSlug]/root-keys", { workspaceSlug });
  },
};
