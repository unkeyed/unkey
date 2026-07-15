/**
 * Route builders for the /auth area. These routes live outside the workspace
 * hierarchy, so the builders take no scope. The sign-in route is an optional
 * catch-all ([[...sign-in]]); buildRoute drops the segment to yield the base
 * /auth/sign-in path.
 */
import type { Route } from "next";
import { buildRoute } from "./shared";

export const authRoutes = {
  signIn(): Route {
    return buildRoute("/auth/sign-in/[[...sign-in]]", {});
  },
};
