"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { useLiveQuery } from "@tanstack/react-db";
import { Cube, Plus } from "@unkey/icons";
import { Crumb } from "./crumb";
import type { CrumbPopoverItem } from "./crumb-popover";

export function ProjectCrumb({ projectSlug }: { projectSlug: string }) {
  const workspace = useWorkspaceNavigation();
  const projectsQuery = useLiveQuery((q) =>
    q.from({ project: collection.projects }).select(({ project }) => ({
      id: project.id,
      name: project.name,
      slug: project.slug,
    })),
  );
  const projects = projectsQuery.data ?? [];
  const current = projects.find((p) => p.slug === projectSlug);
  const loading = projectsQuery.isLoading;

  const items: CrumbPopoverItem[] = projects.map((p) => ({
    id: p.slug,
    label: p.name,
    href: `/${workspace.slug}/projects/${p.slug}/apps`,
  }));

  return (
    <Crumb
      icon={<Cube className="size-3.5 text-accent-11" iconSize="sm-regular" />}
      label={current?.name ?? projectSlug}
      loading={loading}
      href={`/${workspace.slug}/projects/${projectSlug}/apps`}
      items={items}
      currentId={projectSlug}
      searchPlaceholder="Find project..."
      emptyText="No projects found"
      footer={{
        icon: Plus,
        label: "New project",
        href: `/${workspace.slug}/projects/new`,
      }}
    />
  );
}
