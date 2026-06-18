/**
 * Route builders for workspace-level routes that live outside the
 * /[workspaceSlug] hierarchy, so the builders take no scope. `create` is the
 * onboarding entry (/new) used to create the first or a new workspace.
 */
import type { Route } from "next";
import { buildRoute } from "./shared";

export const workspaceRoutes = {
  create(): Route {
    return buildRoute("/new", {});
  },
};
