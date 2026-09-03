import type { Route } from "next";
import { type WorkspaceScope, buildRoute } from "./shared";

type AlertScope = WorkspaceScope & { alertId: string };

export const alertRoutes = {
  list({ workspaceSlug }: WorkspaceScope): Route {
    return buildRoute("/[workspaceSlug]/alerts", { workspaceSlug });
  },

  detail({ workspaceSlug, alertId }: AlertScope): Route {
    return buildRoute("/[workspaceSlug]/alerts/[alertId]", { workspaceSlug, alertId });
  },
};
