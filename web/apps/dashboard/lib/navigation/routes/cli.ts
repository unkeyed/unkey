/**
 * Route builders for the /cli area. These routes live outside the workspace
 * hierarchy, so the builders take no scope.
 */
import type { Route } from "next";
import { buildRoute } from "./shared";

export const cliRoutes = {
  sessions(): Route {
    return buildRoute("/cli/sessions", {});
  },
};
