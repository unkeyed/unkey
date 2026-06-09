"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { projectPath, projectsPath } from "@/lib/navigation/routes";
import { useLiveQuery } from "@tanstack/react-db";
import { Cube, Plus } from "@unkey/icons";
import { Crumb } from "./crumb";
import type { CrumbPopoverItem } from "./crumb-popover";

export function ProjectCrumb({ projectId }: { projectId: string }) {
  const workspace = useWorkspaceNavigation();
  const projectsQuery = useLiveQuery((q) =>
    q.from({ project: collection.projects }).select(({ project }) => ({
      id: project.id,
      name: project.name,
    })),
  );
  const projects = projectsQuery.data ?? [];
  const current = projects.find((p) => p.id === projectId);
  const loading = projectsQuery.isLoading;

  const items: CrumbPopoverItem[] = projects.map((p) => ({
    id: p.id,
    label: p.name,
    href: projectPath({ workspaceSlug: workspace.slug, projectId: p.id }),
  }));

  return (
    <Crumb
      icon={<Cube className="size-3.5 text-accent-11" iconSize="sm-regular" />}
      label={current?.name ?? projectId}
      loading={loading}
      href={projectPath({ workspaceSlug: workspace.slug, projectId })}
      items={items}
      currentId={projectId}
      searchPlaceholder="Find project..."
      emptyText="No projects found"
      footer={{
        icon: Plus,
        label: "New project",
        href: projectsPath({ workspaceSlug: workspace.slug, new: true }),
      }}
    />
  );
}
