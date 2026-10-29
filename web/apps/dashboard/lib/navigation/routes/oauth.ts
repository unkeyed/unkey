/**
 * Route builders for the /oauth area. These routes live outside the workspace
 * hierarchy, so the builders take no scope.
 */
import type { Route } from "next";
import { buildRoute } from "./shared";

export const oauthRoutes = {
  consent(query?: { user_code?: string }): Route {
    return buildRoute("/oauth/consent", {}, query);
  },
};
