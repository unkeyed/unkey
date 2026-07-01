/**
 * Single entrypoint for area route builders. Import `routes` and call through
 * the hierarchy: `routes.projects.apps.deployment(scope)`. New areas register
 * here as they get builders.
 */
import { apiRoutes } from "./apis";
import { auditRoutes } from "./audit";
import { authRoutes } from "./auth";
import { authorizationRoutes } from "./authorization";
import { identityRoutes } from "./identities";
import { logRoutes } from "./logs";
import { projectRoutes } from "./projects";
import { ratelimitRoutes } from "./ratelimits";
import { settingsRoutes } from "./settings";
import { workspaceRoutes } from "./workspaces";

export { buildRoute } from "./shared";
export type { CheckoutIntent, DeployCheckoutOrigin, DeployCheckoutPlan } from "./settings";

export const routes = {
  projects: projectRoutes,
  ratelimits: ratelimitRoutes,
  settings: settingsRoutes,
  apis: apiRoutes,
  authorization: authorizationRoutes,
  identities: identityRoutes,
  audit: auditRoutes,
  logs: logRoutes,
  auth: authRoutes,
  workspaces: workspaceRoutes,
};
