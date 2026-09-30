"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { PRODUCTION_ENVIRONMENT_SLUG } from "@/lib/collections/deploy/environments";
import { routes } from "@/lib/navigation/routes";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { Github, IconTerminalOutline18 } from "@unkey/icons";
import { CrumbLink } from "./crumb";
import { EnvironmentSwitcher } from "./environment-switcher";

type AppCrumbProps = {
  projectId: string;
  appId: string;
  environmentSlug?: string;
};

// Apps are switched from the project page; the only switcher at this level
// is the environment one after the pill.
export function AppCrumb({ projectId, appId, environmentSlug }: AppCrumbProps) {
  const workspace = useWorkspaceNavigation();
  const appQuery = useLiveQuery(
    (q) =>
      q
        .from({ app: collection.apps })
        .where(({ app }) => and(eq(app.projectId, projectId), eq(app.id, appId))),
    [projectId, appId],
  );
  const app = appQuery.data?.at(0);

  return (
    <div className="flex min-w-0 items-center gap-0.5">
      <CrumbLink
        icon={
          app?.repositoryFullName ? (
            <Github className="size-3.5 text-gray-11" />
          ) : (
            <IconTerminalOutline18 className="size-3.5 text-gray-11" />
          )
        }
        label={app?.name ?? appId}
        loading={appQuery.isLoading}
        href={routes.projects.apps.overview({
          workspaceSlug: workspace.slug,
          projectId,
          appId,
          environmentSlug: environmentSlug ?? PRODUCTION_ENVIRONMENT_SLUG,
        })}
      />
      {environmentSlug && (
        <EnvironmentSwitcher
          projectId={projectId}
          appId={appId}
          environmentSlug={environmentSlug}
        />
      )}
    </div>
  );
}
