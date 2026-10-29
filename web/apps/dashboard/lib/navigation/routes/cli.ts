/**
 * Route builders for the /cli area. These routes live outside the workspace
 * hierarchy, so the builders take no scope.
 */
import type { Route } from "next";
import { buildRoute } from "./shared";

export const cliRoutes = {
  device(query?: { user_code?: string }): Route {
    return buildRoute("/cli/device", {}, query);
  },
};
