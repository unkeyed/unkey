import { type WorkspaceScope, buildRoute } from "./shared";

export const alertRoutes = {
  list({ workspaceSlug }: WorkspaceScope) {
    return buildRoute("/[workspaceSlug]/alerts", { workspaceSlug });
  },
};
