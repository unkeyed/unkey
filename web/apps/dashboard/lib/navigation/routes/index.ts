/**
 * Single entrypoint for area route builders. Import `routes` and call through
 * the hierarchy: `routes.projects.apps.deployment(scope)`. New areas register
 * here as they get builders.
 */
import { projectRoutes } from "./projects";
import { ratelimitRoutes } from "./ratelimits";
import { settingsRoutes } from "./settings";

export { buildRoute } from "./shared";

export const routes = {
  projects: projectRoutes,
  ratelimits: ratelimitRoutes,
  settings: settingsRoutes,
};
