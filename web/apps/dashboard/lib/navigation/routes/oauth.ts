import type { Route } from "next";
import { buildRoute } from "./shared";

export const oauthRoutes = {
  consent(query?: { user_code?: string }): Route {
    return buildRoute("/oauth/consent", {}, query);
  },
};
