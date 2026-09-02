"use client";

import { useLiveQuery } from "@tanstack/react-db";
import { IconCubeOutline18, IconPlusOutline18 } from "nucleo-ui-outline-18";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { routes } from "@/lib/navigation/routes";
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
    href: routes.projects.detail({ workspaceSlug: workspace.slug, projectId: p.id }),
  }));

  return (
    <Crumb
      icon={<IconCubeOutline18 className="size-3.5 text-accent-11" />}
      label={current?.name ?? projectId}
      loading={loading}
      href={routes.projects.detail({ workspaceSlug: workspace.slug, projectId })}
      items={items}
      currentId={projectId}
      searchPlaceholder="Find project..."
      emptyText="No projects found"
      footer={{
        icon: IconPlusOutline18,
        label: "New project",
        href: routes.projects.list({ workspaceSlug: workspace.slug, new: true }),
      }}
    />
  );
}
