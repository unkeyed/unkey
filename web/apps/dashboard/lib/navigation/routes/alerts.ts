import { type WorkspaceScope, buildRoute } from "./shared";

export const alertRoutes = {
  list({ workspaceSlug }: WorkspaceScope) {
    return buildRoute("/[workspaceSlug]/alerts", { workspaceSlug });
  },
  detail({ workspaceSlug, alertId }: WorkspaceScope & { alertId: string }) {
    return buildRoute("/[workspaceSlug]/alerts/[alertId]", { workspaceSlug, alertId });
  },
};
