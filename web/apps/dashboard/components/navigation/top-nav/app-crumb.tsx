"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { PRODUCTION_ENVIRONMENT_SLUG } from "@/lib/collections/deploy/environments";
import { routes } from "@/lib/navigation/routes";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { Github, IconPlusOutline18, IconTerminalOutline18 } from "@unkey/icons";
import { Crumb } from "./crumb";
import type { CrumbPopoverItem } from "./crumb-popover";
import { EnvironmentSwitcher } from "./environment-switcher";

type AppCrumbProps = {
  projectId: string;
  appId: string;
  environmentSlug?: string;
};

export function AppCrumb({ projectId, appId, environmentSlug }: AppCrumbProps) {
  const workspace = useWorkspaceNavigation();
  const appsQuery = useLiveQuery(
    (q) => q.from({ app: collection.apps }).where(({ app }) => eq(app.projectId, projectId)),
    [projectId],
  );
  const environmentsQuery = useLiveQuery(
    (q) =>
      q.from({ env: collection.environments }).where(({ env }) => eq(env.projectId, projectId)),
    [projectId],
  );
  const apps = appsQuery.data ?? [];
  const environments = environmentsQuery.data ?? [];
  const current = apps.find((a) => a.id === appId);

  // Stay in the same environment across apps when the target has one by that
  // slug; otherwise fall back to production.
  const hrefFor = (targetAppId: string) =>
    routes.projects.apps.overview({
      workspaceSlug: workspace.slug,
      projectId,
      appId: targetAppId,
      environmentSlug:
        environmentSlug &&
        environments.some((e) => e.appId === targetAppId && e.slug === environmentSlug)
          ? environmentSlug
          : PRODUCTION_ENVIRONMENT_SLUG,
    });

  const items: CrumbPopoverItem[] = apps.map((a) => ({
    id: a.id,
    label: a.name,
    href: hrefFor(a.id),
  }));

  return (
    <Crumb
      icon={
        current?.repositoryFullName ? (
          <Github className="size-3.5 text-gray-11" />
        ) : (
          <IconTerminalOutline18 className="size-3.5 text-gray-11" />
        )
      }
      label={current?.name ?? appId}
      loading={appsQuery.isLoading}
      href={hrefFor(appId)}
      items={items}
      currentId={appId}
      badge={
        environmentSlug ? (
          <EnvironmentSwitcher
            projectId={projectId}
            appId={appId}
            environmentSlug={environmentSlug}
          />
        ) : undefined
      }
      searchPlaceholder="Find app..."
      emptyText="No apps found"
      footer={{
        icon: IconPlusOutline18,
        label: "New app",
        href: routes.projects.apps.new({ workspaceSlug: workspace.slug, projectId }),
      }}
    />
  );
}
