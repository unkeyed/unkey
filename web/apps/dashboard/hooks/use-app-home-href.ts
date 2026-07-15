"use client";

import { useFlag } from "@/lib/flags/provider";
import { routes } from "@/lib/navigation/routes";

type AppScope = { workspaceSlug: string; projectId: string; appId: string };

export function useAppHomeHref() {
  const appOverview = useFlag("appOverview");
  return (scope: AppScope) =>
    appOverview ? routes.projects.apps.overview(scope) : routes.projects.apps.deployments(scope);
}
