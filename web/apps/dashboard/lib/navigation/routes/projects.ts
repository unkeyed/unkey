/**
 * Route builders for the /projects area, exposed as one nested object so call
 * sites read like the url hierarchy: `routes.projects.apps.settings(scope)`.
 * Every navigable result goes through buildRoute, which checks the bracket
 * pattern against Next's generated route table (typedRoutes) and types the
 * params from the generated ParamMap.
 */
import type { Route } from "next";
import type { QueryParams } from "../url";
import type { DeployCheckoutOrigin, DeployCheckoutPlan } from "./settings";
import { type WorkspaceScope, buildRoute } from "./shared";

type ProjectScope = WorkspaceScope & { projectId: string };
export type AppScope = ProjectScope & { appId: string; environmentSlug: string };

/**
 * Page segments under /apps/[appId]/[environmentSlug]. The environment layout
 * treats these as reserved so a pre-environment url like /apps/x/settings
 * redirects instead of resolving "settings" as an environment slug.
 */
export const APP_PAGES = [
  "overview",
  "deployments",
  "env-vars",
  "policies",
  "settings",
  "openapi-diff",
] as const;
export type AppPage = (typeof APP_PAGES)[number];

export function isAppPage(segment: string): segment is AppPage {
  return APP_PAGES.some((page) => page === segment);
}

const APP_ROOT = "/[workspaceSlug]/projects/[projectId]/apps/[appId]/[environmentSlug]";

export const projectRoutes = {
  list({ workspaceSlug, new: isNew }: WorkspaceScope & { new?: boolean }): Route {
    return buildRoute("/[workspaceSlug]/projects", { workspaceSlug }, { new: isNew || undefined });
  },

  // Compute-plan gate hand-off: the projects landing reads these params,
  // subscribes the chosen plan (card already on file), and on `from=create`
  // opens the create-project dialog.
  pendingSubscribe({
    workspaceSlug,
    plan,
    from,
  }: WorkspaceScope & { plan: DeployCheckoutPlan; from: DeployCheckoutOrigin }): Route {
    return buildRoute("/[workspaceSlug]/projects", { workspaceSlug }, { pendingPlan: plan, from });
  },

  detail(scope: ProjectScope): Route {
    return buildRoute("/[workspaceSlug]/projects/[projectId]", projectParams(scope));
  },

  overview(scope: ProjectScope): Route {
    return buildRoute("/[workspaceSlug]/projects/[projectId]/overview", projectParams(scope));
  },

  settings(scope: ProjectScope): Route {
    return buildRoute("/[workspaceSlug]/projects/[projectId]/settings", projectParams(scope));
  },

  logs({
    appId,
    deploymentId,
    ...scope
  }: ProjectScope & { appId?: string; deploymentId?: string }): Route {
    return buildRoute("/[workspaceSlug]/projects/[projectId]/logs", projectParams(scope), {
      appId: appId ? isFilter(appId) : undefined,
      deploymentId: deploymentId ? isFilter(deploymentId) : undefined,
    });
  },

  requests({
    since,
    appId,
    deploymentId,
    ...scope
  }: ProjectScope & { since?: string; appId?: string; deploymentId?: string }): Route {
    return buildRoute("/[workspaceSlug]/projects/[projectId]/requests", projectParams(scope), {
      since,
      appId: appId ? isFilter(appId) : undefined,
      deploymentId: deploymentId ? isFilter(deploymentId) : undefined,
    });
  },

  apps: {
    new({ step, appId, ...scope }: ProjectScope & { step?: string; appId?: string }): Route {
      return buildRoute("/[workspaceSlug]/projects/[projectId]/apps/new", projectParams(scope), {
        step,
        appId,
      });
    },

    overview(scope: AppScope): Route {
      return appPage("overview", scope);
    },

    settings(scope: AppScope): Route {
      return appPage("settings", scope);
    },

    envVars(scope: AppScope): Route {
      return appPage("env-vars", scope);
    },

    policies(scope: AppScope): Route {
      return appPage("policies", scope);
    },

    deployments(scope: AppScope): Route {
      return appPage("deployments", scope);
    },

    deployment({
      deploymentId,
      build,
      ...scope
    }: AppScope & { deploymentId: string; build?: boolean }): Route {
      return buildRoute(
        `${APP_ROOT}/deployments/[deploymentId]`,
        { ...appParams(scope), deploymentId },
        { build: build || undefined },
      );
    },

    openapiDiff({ from, to, ...scope }: AppScope & { from?: string; to?: string }): Route {
      return appPage("openapi-diff", scope, { from, to });
    },
  },
};

function projectParams({ workspaceSlug, projectId }: ProjectScope) {
  return { workspaceSlug, projectId };
}

function appParams({ appId, environmentSlug, ...scope }: AppScope) {
  return { ...projectParams(scope), appId, environmentSlug };
}

function appPage(page: AppPage, scope: AppScope, query?: QueryParams): Route {
  return buildRoute(`${APP_ROOT}/${page}`, appParams(scope), query);
}

/**
 * `is:` is the logs/requests table filter syntax for a deployment id. Both
 * backends match the id exactly, and the deployment filter UI emits `is`,
 * so links use the same operator to keep filter chips consistent.
 */
function isFilter(value: string): `is:${string}` {
  return `is:${value}`;
}
