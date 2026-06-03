"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { Layers3, Plus } from "@unkey/icons";
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
    href: `/${workspace.slug}/projects/${projectId}/apps/${a.id}/deployments`,
  }));

  return (
    <Crumb
      icon={<Layers3 className="size-3.5 text-accent-11" iconSize="sm-regular" />}
      label={current?.name ?? appId}
      loading={appsQuery.isLoading}
      href={`/${workspace.slug}/projects/${projectId}/apps/${appId}/deployments`}
      items={items}
      currentId={appId}
      searchPlaceholder="Find app..."
      emptyText="No apps found"
      footer={{
        icon: Plus,
        label: "New app",
        href: `/${workspace.slug}/projects/${projectId}/apps/new`,
      }}
    />
  );
}
