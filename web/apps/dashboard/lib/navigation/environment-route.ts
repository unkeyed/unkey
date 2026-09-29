import { ENVIRONMENT_KIND, type Environment } from "@/lib/collections/deploy/environments";
import type { Route } from "next";
import { type AppPage, isAppPage } from "./routes/projects";

/**
 * How the `[environmentSlug]` segment under /apps/[appId] resolved.
 *
 * - `legacy`: the segment is a page name from before environments lived in the
 *   url (/apps/x/settings), so the default slug is inserted in front of it.
 * - `unknown`: the segment matches no environment, so it is replaced.
 */
export type EnvironmentRoute =
  | { kind: "ok"; environment: Environment }
  | { kind: "legacy"; redirectSlug: string }
  | { kind: "unknown"; redirectSlug: string }
  | { kind: "notFound" };

export const APP_HOME_PAGE: AppPage = "overview";

export function defaultEnvironment(environments: Environment[]): Environment | undefined {
  return (
    environments.find((environment) => environment.kind === ENVIRONMENT_KIND.production) ??
    environments.at(0)
  );
}

export function resolveEnvironmentRoute(
  param: string,
  environments: Environment[],
): EnvironmentRoute {
  const fallback = defaultEnvironment(environments);
  if (isAppPage(param)) {
    return fallback ? { kind: "legacy", redirectSlug: fallback.slug } : { kind: "notFound" };
  }
  const environment = environments.find((candidate) => candidate.slug === param);
  if (environment) {
    return { kind: "ok", environment };
  }
  return fallback ? { kind: "unknown", redirectSlug: fallback.slug } : { kind: "notFound" };
}

/**
 * Same page, other environment: swaps the segment after /apps/[appId]. A path
 * that ends at the environment lands on the app home page.
 */
export function withEnvironmentSlug(pathname: string, appId: string, slug: string): Route {
  const [base, rest] = splitAtApp(pathname, appId);
  const tail = rest.replace(/^\/[^/]*/, "");
  return widen(`${base}/${slug}${tail || `/${APP_HOME_PAGE}`}`);
}

export function environmentRedirectPath(
  pathname: string,
  appId: string,
  route: Extract<EnvironmentRoute, { kind: "legacy" | "unknown" }>,
): Route {
  if (route.kind === "legacy") {
    const [base, rest] = splitAtApp(pathname, appId);
    return widen(`${base}/${route.redirectSlug}${rest}`);
  }
  return withEnvironmentSlug(pathname, appId, route.redirectSlug);
}

// Bare Route cannot express a rewritten pathname. The input came from the
// router and only the environment segment changes, so this is the same
// widening buildRoute makes for dynamic segments.
function widen(pathname: string): Route {
  return pathname as Route;
}

function splitAtApp(pathname: string, appId: string): [base: string, rest: string] {
  const marker = `/apps/${appId}`;
  const [before, after = ""] = pathname.split(marker, 2);
  return [`${before}${marker}`, after];
}
