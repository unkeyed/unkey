/**
 * Single entrypoint for area route builders. Import `routes` and call through
 * the hierarchy: `routes.projects.apps.deployment(scope)`. New areas (apis,
 * ratelimits) register here as they get builders.
 */
import { apiRoutes } from "./apis";
import { projectRoutes } from "./projects";

export { buildRoute } from "./shared";

export const routes = {
  projects: projectRoutes,
  apis: apiRoutes,
};
