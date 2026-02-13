"use client";
import type { NavItem } from "@/components/navigation/sidebar/workspace-navigations";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { shortenId } from "@/lib/shorten-id";
import { eq, useLiveQuery } from "@tanstack/react-db";
import {
  ArrowOppositeDirectionY,
  Cube,
  Gear,
  InputSearch,
  Layers3,
  Nodes,
} from "@unkey/icons";
import { useMemo } from "react";

type ProjectScopeResult =
  | { isInsideProject: false }
  | {
    isInsideProject: true;
    projectId: string;
    projectName: string;
    backHref: string;
    navItems: NavItem[];
  };

export const useProjectScopedNavigation = (segments: string[]): ProjectScopeResult => {
  const workspace = useWorkspaceNavigation();
  const cleanSegments = segments.filter((s) => !s.startsWith("("));
  const projectsIndex = cleanSegments.findIndex((s) => s === "projects");
  const projectId = projectsIndex !== -1 ? cleanSegments.at(projectsIndex + 1) : undefined;

  const { data: projectData } = useLiveQuery((q) =>
    q.from({ project: collection.projects }).orderBy(({ project }) => project.id, "desc"),
  );

  const safeProjectId = projectId ?? "__none__";

  const { data: deploymentData } = useLiveQuery((q) =>
    q
      .from({ deployment: collection.deployments })
      .where(({ deployment }) => eq(deployment.projectId, safeProjectId))
      .orderBy(({ deployment }) => deployment.createdAt, "desc"),
  );

  return useMemo(() => {
    if (!projectId || projectsIndex === -1) {
      return { isInsideProject: false };
    }

    const project = projectData?.find((p) => p.id === projectId);
    const projectName = project?.name ?? projectId;
    const basePath = `/${workspace.slug}/projects`;
    const currentSegment = cleanSegments.at(projectsIndex + 2);

    // Parse deployment context
    const isOnDeployments = currentSegment === "deployments";
    const deploymentId = isOnDeployments ? cleanSegments.at(projectsIndex + 3) : undefined;
    const deploymentSubSegment = deploymentId
      ? cleanSegments.at(projectsIndex + 4)
      : undefined;

    // Build deployment nested children when a specific deployment is selected
    const deploymentItems: NavItem[] = [];
    if (deploymentId) {
      const deploymentPath = `${basePath}/${projectId}/deployments/${deploymentId}`;

      deploymentItems.push(
        {
          icon: Cube,
          href: deploymentPath,
          label: "Overview",
          active: deploymentSubSegment === undefined,
          tooltip: "Overview",
        },
        {
          icon: Nodes,
          href: `${deploymentPath}/network`,
          label: "Network",
          active: deploymentSubSegment === "network",
          tooltip: "Network",
        },
      );
    }

    const navItems: NavItem[] = [
      {
        icon: Cube,
        href: `${basePath}/${projectId}`,
        label: "Overview",
        active: currentSegment === undefined,
        tooltip: "Overview",
      },
      {
        icon: Layers3,
        href: `${basePath}/${projectId}/deployments`,
        label: "Deployments",
        active: isOnDeployments,
        tooltip: "Deployments",
        showSubItems: deploymentId !== undefined,
        items: deploymentItems.length > 0 ? deploymentItems : undefined,
      },
      {
        icon: ArrowOppositeDirectionY,
        href: `${basePath}/${projectId}/requests`,
        label: "Requests",
        active: currentSegment === "requests",
        tooltip: "Requests",
      },
      {
        icon: InputSearch,
        href: `${basePath}/${projectId}/logs`,
        label: "Logs",
        active: currentSegment === "logs",
        tooltip: "Logs",
      },
      {
        icon: Gear,
        href: `${basePath}/${projectId}/settings`,
        label: "Settings",
        active: currentSegment === "settings",
        tooltip: "Settings",
      },
    ];

    return {
      isInsideProject: true,
      projectId,
      projectName,
      backHref: basePath,
      navItems,
    };
  }, [projectId, projectsIndex, projectData, deploymentData, workspace.slug, cleanSegments]);
};
