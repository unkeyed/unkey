"use client";
import type { NavItem } from "@/components/navigation/sidebar/workspace-navigations";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { useLiveQuery } from "@tanstack/react-db";
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
      const currentProjectActive = segments.at(0) === "projects" && segments.at(1) === project.id;

      const projectNavItem: NavItem = {
        href: `${basePath}/${project.id}`,
        icon: null,
        label: project.name,
        active: currentProjectActive,
        showSubItems: true,
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
