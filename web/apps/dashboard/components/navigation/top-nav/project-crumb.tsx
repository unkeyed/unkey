"use client";

import type { ProjectOwner } from "@/hooks/use-resource-project-id";
import { useVisibleProjects } from "@/hooks/use-visible-projects";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { projectDisplayName } from "@/lib/collections/deploy/projects";
import { routes } from "@/lib/navigation/routes";
import { IconCubeOutline18, IconPlusOutline18 } from "@unkey/icons";
import { Crumb } from "./crumb";
import type { CrumbPopoverItem } from "./crumb-popover";

export function ProjectCrumb({ owner }: { owner: ProjectOwner }) {
  const workspace = useWorkspaceNavigation();
  const projectsQuery = useVisibleProjects();
  const projects = projectsQuery.data ?? [];
  const projectId = owner.state === "resolved" ? owner.projectId : null;
  const current = projects.find((p) => p.id === projectId);
  const loading = owner.state === "loading" || projectsQuery.isLoading;

  const items: CrumbPopoverItem[] = projects.map((p) => ({
    id: p.id,
    label: projectDisplayName(p, workspace.name),
    href: routes.projects.overview({ workspaceSlug: workspace.slug, projectId: p.id }),
  }));

  return (
    <Crumb
      icon={<IconCubeOutline18 className="size-3.5 text-gray-11" />}
      label={current ? projectDisplayName(current, workspace.name) : (projectId ?? "Project")}
      loading={loading}
      href={
        projectId
          ? routes.projects.overview({ workspaceSlug: workspace.slug, projectId })
          : routes.projects.list({ workspaceSlug: workspace.slug })
      }
      items={items}
      currentId={projectId ?? ""}
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
