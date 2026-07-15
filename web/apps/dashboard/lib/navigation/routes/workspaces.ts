/**
 * Route builders for workspace-level routes that sit at or above the
 * /[workspaceSlug] hierarchy. `root` is the app root (resolves to the
 * last-used workspace), `overview` is a workspace home, and `create` is the
 * onboarding entry (/new) used to create the first or a new workspace.
 */
import type { Route } from "next";
import { type WorkspaceScope, buildRoute } from "./shared";

export const workspaceRoutes = {
  root(): Route {
    return buildRoute("/", {});
  },

  overview({ workspaceSlug }: WorkspaceScope): Route {
    return buildRoute("/[workspaceSlug]", { workspaceSlug });
  },

  create(): Route {
    return buildRoute("/new", {});
  },
};
