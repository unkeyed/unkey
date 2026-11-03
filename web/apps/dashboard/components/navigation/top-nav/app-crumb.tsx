"use client";

import { useAppHomeHref } from "@/hooks/use-app-home-href";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { useProjectApps } from "@/lib/queries/apps";
import { Github, IconPlusOutline18, IconTerminalOutline18 } from "@unkey/icons";
import { Crumb } from "./crumb";
import type { CrumbPopoverItem } from "./crumb-popover";

export function AppCrumb({ projectId, appId }: { projectId: string; appId: string }) {
  const workspace = useWorkspaceNavigation();
  const appHomeHref = useAppHomeHref();
  const appsQuery = useProjectApps(projectId);
  const apps = appsQuery.data ?? [];
  const current = apps.find((a) => a.id === appId);

  const items: CrumbPopoverItem[] = apps.map((a) => ({
    id: a.id,
    label: a.name,
    href: appHomeHref({
      workspaceSlug: workspace.slug,
      projectId,
      appId: a.id,
    }),
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
      href={appHomeHref({ workspaceSlug: workspace.slug, projectId, appId })}
      items={items}
      currentId={appId}
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
