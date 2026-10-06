"use client";

import { latestDeploymentId } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/new/wizard-model";
import type { App } from "@/lib/collections/deploy/apps";
import { routes } from "@/lib/navigation/routes";

type ProjectScope = { workspaceSlug: string; projectId: string };
type HomeApp = Pick<App, "id" | "headlineDeployment" | "currentDeploymentId">;

// An app that never deployed has no overview to show; its home is the new-app
// flow, resumed where it left off.
export function useAppHomeHref() {
  return (scope: ProjectScope, app: HomeApp) =>
    latestDeploymentId(app) === null
      ? routes.projects.apps.new({ ...scope, appId: app.id })
      : routes.projects.apps.overview({ ...scope, appId: app.id });
}
