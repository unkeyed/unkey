"use client";
import type { NavItem } from "@/components/navigation/sidebar/workspace-navigations";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { useLiveQuery } from "@tanstack/react-db";
import { Cloud, Connections, GridCircle, Layers3 } from "@unkey/icons";
import { useSelectedLayoutSegments } from "next/navigation";
import { useMemo } from "react";

export const useProjectNavigation = (baseNavItems: NavItem[]) => {
  const segments = useSelectedLayoutSegments() ?? [];
  const workspace = useWorkspaceNavigation();

  const { data, isLoading } = useLiveQuery((q) =>
    q.from({ project: collection.projects }).orderBy(({ project }) => project.id, "desc"),
  );

  const basePath = `/${workspace.slug}/projects`;
  const projectNavItems = useMemo(() => {
    if (!data) {
      return [];
    }
    return data.map((project) => {
      const pIndex = segments.findIndex((s) => s === "projects");
      const currentProjectActive = pIndex !== -1 && segments.at(pIndex + 1) === project.id;

      // Create sub-items
      const subItems: NavItem[] = [
        {
          icon: GridCircle,
          href: `${basePath}/${project.id}`,
          label: "Overview",
          active: currentProjectActive && !segments.at(pIndex + 2),
        },
        {
          icon: Cloud,
          href: `${basePath}/${project.id}/deployments`,
          label: "Deployments",
          active: currentProjectActive && segments.at(pIndex + 2) === "deployments",
        },
        {
          icon: Layers3,
          href: `${basePath}/${project.id}/sentinel-logs`,
          label: "Sentinel Logs",
          active: currentProjectActive && segments.at(pIndex + 2) === "sentinel-logs",
        },
        {
          icon: Connections,
          href: `${basePath}/${project.id}/openapi-diff`,
          label: "Open API Diff",
          active: currentProjectActive && segments.at(pIndex + 2) === "openapi-diff",
        },
      ];

      const projectNavItem: NavItem = {
        href: `${basePath}/${project.id}`,
        icon: null,
        label: project.name,
        active: currentProjectActive,
        showSubItems: true,
        items: subItems,
      };

      return projectNavItem;
    });
  }, [data, segments, basePath]);

  const enhancedNavItems = useMemo(() => {
    const items = [...baseNavItems];
    const projectsItemIndex = items.findIndex((item) => item.href === basePath);

    if (projectsItemIndex !== -1) {
      const projectsItem = { ...items[projectsItemIndex] };
      projectsItem.showSubItems = true;
      projectsItem.items = [...(projectsItem.items || []), ...projectNavItems];

      items[projectsItemIndex] = projectsItem;
    }

    return items;
  }, [baseNavItems, projectNavItems, basePath]);

  return {
    enhancedNavItems,
    isLoading,
  };
};
