"use client";

import { routes } from "@/lib/navigation/routes";

type AppScope = { workspaceSlug: string; projectId: string; appId: string };

export function useAppHomeHref() {
  return (scope: AppScope) => routes.projects.apps.overview(scope);
}
