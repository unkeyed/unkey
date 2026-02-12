"use client";
import type { NavItem } from "@/components/navigation/sidebar/workspace-navigations";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { useLiveQuery } from "@tanstack/react-db";
import { Cloud, Gear, GridCircle } from "@unkey/icons";
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
      const projectsSegmentIndex = segments.findIndex((s) => s === "projects");

      const projectIdIndex = projectsSegmentIndex + 1;
      const subRouteIndex = projectsSegmentIndex + 3;

      const currentProjectActive =
        projectsSegmentIndex !== -1 && segments.at(projectIdIndex) === project.id;

      const currentSubRoute = segments.at(subRouteIndex);

      // Create sub-items
      const subItems: NavItem[] = [
        {
          icon: GridCircle,
          href: `${basePath}/${project.id}`,
          label: "Overview",
          active: currentProjectActive && !currentSubRoute,
        },
        {
          icon: Cloud,
          href: `${basePath}/${project.id}/deployments`,
          label: "Deployments",
          active: currentProjectActive && currentSubRoute === "deployments",
        },
        {
          icon: Gear,
          href: `${basePath}/${project.id}/settings`,
          label: "Settings",
          active: currentProjectActive && currentSubRoute === "settings",
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
