"use client";

import { PRODUCTION_ENVIRONMENT_SLUG } from "@/lib/collections/deploy/environments";
import { routes } from "@/lib/navigation/routes";

type AppScope = { workspaceSlug: string; projectId: string; appId: string };

export function useAppHomeHref() {
  return (scope: AppScope) =>
    routes.projects.apps.overview({ ...scope, environmentSlug: PRODUCTION_ENVIRONMENT_SLUG });
}
