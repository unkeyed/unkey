"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { Github, Plus, Terminal } from "@unkey/icons";
import { Crumb } from "./crumb";
import type { CrumbPopoverItem } from "./crumb-popover";

export function AppCrumb({ projectSlug, appSlug }: { projectSlug: string; appSlug: string }) {
  const workspace = useWorkspaceNavigation();
  const basePath = `/${workspace.slug}/projects/${projectSlug}`;

  const appsQuery = useLiveQuery(
    (q) => q.from({ app: collection.apps }).where(({ app }) => eq(app.projectSlug, projectSlug)),
    [projectSlug],
  );
  const apps = appsQuery.data ?? [];
  const current = apps.find((a) => a.slug === appSlug);

  const items: CrumbPopoverItem[] = apps.map((a) => ({
    id: a.id,
    label: a.name,
    href: `${basePath}/apps/${a.slug}/deployments`,
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
      label={current?.name ?? appSlug}
      loading={appsQuery.isLoading}
      href={`${basePath}/apps/${appSlug}/deployments`}
      items={items}
      currentId={current?.id ?? appSlug}
      searchPlaceholder="Find app..."
      emptyText="No apps found"
      footer={{
        icon: Plus,
        label: "New app",
        href: `${basePath}/apps/new`,
      }}
    />
  );
}
