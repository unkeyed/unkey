"use client";

import { useResolvedProject } from "@/hooks/use-resolved-project";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { Github, Plus, Terminal } from "@unkey/icons";
import { Crumb } from "./crumb";
import type { CrumbPopoverItem } from "./crumb-popover";

export function AppCrumb({ projectSlug, appSlug }: { projectSlug: string; appSlug: string }) {
  const workspace = useWorkspaceNavigation();
  const { projectId, isLoading } = useResolvedProject();
  const basePath = `/${workspace.slug}/projects/${projectSlug}`;

  // The apps collection requires a projectId filter, so wait for the slug to
  // resolve before mounting the query.
  if (!projectId) {
    return (
      <Crumb
        icon={<Terminal className="size-3.5 text-accent-11" iconSize="sm-regular" />}
        label={appSlug}
        loading={isLoading}
        href={`${basePath}/apps/${appSlug}/deployments`}
        items={[]}
        currentId={appSlug}
        searchPlaceholder="Find app..."
        emptyText="No apps found"
        footer={{ icon: Plus, label: "New app", href: `${basePath}/apps/new` }}
      />
    );
  }

  return <ResolvedAppCrumb projectId={projectId} basePath={basePath} appSlug={appSlug} />;
}

function ResolvedAppCrumb({
  projectId,
  basePath,
  appSlug,
}: {
  projectId: string;
  basePath: string;
  appSlug: string;
}) {
  const appsQuery = useLiveQuery(
    (q) => q.from({ app: collection.apps }).where(({ app }) => eq(app.projectId, projectId)),
    [projectId],
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
