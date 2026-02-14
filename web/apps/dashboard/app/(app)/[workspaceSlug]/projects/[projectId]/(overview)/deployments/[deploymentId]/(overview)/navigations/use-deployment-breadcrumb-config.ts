"use client";

import type { QuickNavItem } from "@/components/navbar-popover";
import type { Navbar } from "@/components/navigation/navbar";
import { collection } from "@/lib/collections";
import { shortenId } from "@/lib/shorten-id";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useParams } from "next/navigation";
import { useMemo } from "react";
import type { ComponentPropsWithoutRef } from "react";
import { useProjectData } from "../../../../data-provider";
import { useDeployment } from "../../layout-provider";

export type BreadcrumbItem = ComponentPropsWithoutRef<typeof Navbar.Breadcrumbs.Link> & {
  /** Unique identifier for the breadcrumb item */
  id: string;
  /** Internal: determines if this breadcrumb should be rendered */
  shouldRender: boolean;
  /** Optional QuickNav dropdown configuration */
  quickNavConfig?: {
    items: QuickNavItem[];
    activeItemId?: string;
    shortcutKey?: string;
  };
};

export function useDeploymentBreadcrumbConfig(): BreadcrumbItem[] {
  const params = useParams();

  const workspaceSlug = params.workspaceSlug as string;
  const { projectId, project } = useProjectData();
  const { deploymentId } = useDeployment();

  const { data: deployments } = useLiveQuery((q) =>
    q
      .from({ deployment: collection.deployments })
      .where(({ deployment }) => eq(deployment.projectId, projectId))
      .select(({ deployment }) => ({
        id: deployment.id,
        createdAt: deployment.createdAt,
      }))
      .orderBy(({ deployment }) => deployment.createdAt, "desc"),
  );

  const projects = useLiveQuery((q) =>
    q.from({ project: collection.projects }).select(({ project }) => ({
      id: project.id,
      name: project.name,
    })),
  );

  const activeProject = project
    ? { id: project.id, name: project.name, repositoryFullName: project.repositoryFullName }
    : undefined;

  return useMemo(() => {
    const basePath = `/${workspaceSlug}/projects/${projectId}`;

    const deploymentQuickNavItems: QuickNavItem[] = (deployments ?? []).map((d) => ({
      id: d.id,
      label: shortenId(d.id),
      href: `${basePath}/deployments/${d.id}`,
    }));

    return [
      {
        id: "project",
        children: activeProject?.name || projectId,
        href: `${basePath}/${projectId}`,
        isIdentifier: true,
        shouldRender: true,
        active: false,
        isLast: false,
        noop: true,
        className: "flex",
        quickNavConfig: {
          items: projects.data.map((project) => ({
            id: project.id,
            label: project.name,
            href: `${basePath}/${project.id}`,
          })),
          shortcutKey: "N",
        },
      },

      {
        id: "deployment",
        href: `${basePath}/deployments/${deploymentId}`,
        children: shortenId(deploymentId),
        isIdentifier: true,
        shouldRender: true,
        active: false,
        isLast: false,
        noop: true,
        className: "flex",
        quickNavConfig: {
          items: deploymentQuickNavItems,
          activeItemId: deploymentId,
          shortcutKey: "D",
        },
      },
    ];
  }, [workspaceSlug, projectId, deploymentId, deployments, projects, activeProject?.name]);
}
