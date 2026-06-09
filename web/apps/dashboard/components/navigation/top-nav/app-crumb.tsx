"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { appDeploymentsPath, newAppPath } from "@/lib/navigation/routes/projects";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { Github, Plus, Terminal } from "@unkey/icons";
import { Crumb } from "./crumb";
import type { CrumbPopoverItem } from "./crumb-popover";

export function AppCrumb({ projectId, appId }: { projectId: string; appId: string }) {
  const workspace = useWorkspaceNavigation();
  const appsQuery = useLiveQuery(
    (q) => q.from({ app: collection.apps }).where(({ app }) => eq(app.projectId, projectId)),
    [projectId],
  );
  const apps = appsQuery.data ?? [];
  const current = apps.find((a) => a.id === appId);

  const items: CrumbPopoverItem[] = apps.map((a) => ({
    id: a.id,
    label: a.name,
    href: appDeploymentsPath({ workspaceSlug: workspace.slug, projectId, appId: a.id }),
  }));

  return (
    <Crumb
      icon={
        current?.repositoryFullName ? (
          <Github className="size-3.5 text-accent-11" iconSize="sm-regular" />
        ) : (
          <Terminal className="size-3.5 text-accent-11" iconSize="sm-regular" />
        )
      }
      label={current?.name ?? appId}
      loading={appsQuery.isLoading}
      href={appDeploymentsPath({ workspaceSlug: workspace.slug, projectId, appId })}
      items={items}
      currentId={appId}
      searchPlaceholder="Find app..."
      emptyText="No apps found"
      footer={{
        icon: Plus,
        label: "New app",
        href: newAppPath({ workspaceSlug: workspace.slug, projectId }),
      }}
    />
  );
}
