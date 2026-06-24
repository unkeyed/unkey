/**
 * Scope types shared across route-builder areas. Each area (projects, apis,
 * ratelimits) extends WorkspaceScope with its own ids.
 */
import type { AppRoutes, ParamMap } from "@/.next/types/routes";
import type { Route } from "next";
import { type QueryParams, withQuery } from "../url";

export type WorkspaceScope = { workspaceSlug: string };

/**
 * Build an href from a bracket pattern in Next's generated route table.
 * Membership in AppRoutes is exact literal matching, so a typo'd or removed
 * route fails to compile and ParamMap demands exactly the params the route
 * declares. Params are written verbatim (no encoding) per url.ts conventions.
 *
 * Optional catch-all segments ([[...slug]]) are dropped, yielding the base
 * path (e.g. /auth/sign-in/[[...sign-in]] -> /auth/sign-in); builders never
 * pass catch-all values. Required catch-alls ([...slug]) are not handled.
 */
export function buildRoute<P extends AppRoutes>(
  pattern: P,
  params: ParamMap[P],
  query?: QueryParams,
): Route {
  const path = pattern
    .replace(/\/\[\[\.\.\.[\w-]+\]\]/g, "")
    .replace(/\[(\w+)\]/g, (_, key: string) => (params as Record<string, string>)[key]);
  // Bare Route cannot express dynamic-segment hrefs; the pattern check above
  // already validated the shape, so the widening is safe.
  return (query ? withQuery(path, query) : path) as Route;
}
