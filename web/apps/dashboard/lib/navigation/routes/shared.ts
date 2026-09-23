/**
 * Scope types shared across route-builder areas. Each area (projects, apis,
 * ratelimits) extends WorkspaceScope with its own ids.
 */
import type { AppRouteHandlerRoutes, AppRoutes, ParamMap } from "@/.next/types/routes";
import type { Route } from "next";
import { type QueryParams, withQuery } from "../url";

export type WorkspaceScope = { workspaceSlug: string };

export type ResourceScope = WorkspaceScope & { projectId?: string };

/**
 * Build an href from a bracket pattern in Next's generated route table.
 * Membership in the generated page and route-handler tables is exact literal
 * matching, so a typo'd or removed route fails to compile and ParamMap demands
 * exactly the params the route declares. Params are written verbatim (no
 * encoding) per url.ts conventions.
 *
 * Optional catch-all segments ([[...slug]]) are dropped, yielding the base
 * path (e.g. /auth/sign-in/[[...sign-in]] -> /auth/sign-in); builders never
 * pass catch-all values. Required catch-alls ([...slug]) are not handled.
 */
export function buildRoute<P extends AppRoutes | AppRouteHandlerRoutes>(
  pattern: P,
  params: ParamMap[P],
  query?: QueryParams,
): Route {
  return renderPattern(pattern, params, query);
}

/**
 * A page that is served at workspace level and again inside a project, with
 * the same params plus projectId. `project` only accepts a pattern whose
 * ParamMap is exactly the workspace pattern's params plus projectId. That
 * catches a missing or extra id; two project routes with the same params
 * (the list pages) are told apart only by the route tests.
 */
export type ScopedPatterns<W extends AppRoutes> = {
  workspace: W;
  project: ProjectPatternFor<W>;
};

type ProjectPatternFor<W extends AppRoutes> = {
  [P in AppRoutes]: ParamMap[P] extends ParamMap[W] & { projectId: string }
    ? ParamMap[W] & { projectId: string } extends ParamMap[P]
      ? P
      : never
    : never;
}[AppRoutes];

export function scopedRoute<W extends AppRoutes>(
  patterns: ScopedPatterns<W>,
  { projectId, ...params }: ParamMap[W] & { projectId?: string },
  query?: QueryParams,
): Route {
  return projectId
    ? renderPattern(patterns.project, { ...params, projectId }, query)
    : renderPattern(patterns.workspace, params, query);
}

function renderPattern(pattern: string, params: object, query?: QueryParams): Route {
  const values = new Map(Object.entries(params));
  const path = pattern
    .replace(/\/\[\[\.\.\.[\w-]+\]\]/g, "")
    .replace(/\[(\w+)\]/g, (_, key: string) => String(values.get(key)));
  // Bare Route cannot express dynamic-segment hrefs; the typed entry points
  // above already validated the shape, so the widening is safe.
  return (query ? withQuery(path, query) : path) as Route;
}
