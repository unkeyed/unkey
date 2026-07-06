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
import { monetizationRoutes } from "./monetization";
import { projectRoutes } from "./projects";
import { ratelimitRoutes } from "./ratelimits";
import { settingsRoutes } from "./settings";
import { workspaceRoutes } from "./workspaces";

export { buildRoute } from "./shared";

export const routes = {
  projects: projectRoutes,
  ratelimits: ratelimitRoutes,
  settings: settingsRoutes,
  apis: apiRoutes,
  authorization: authorizationRoutes,
  identities: identityRoutes,
  monetization: monetizationRoutes,
  audit: auditRoutes,
  logs: logRoutes,
  auth: authRoutes,
  workspaces: workspaceRoutes,
};
