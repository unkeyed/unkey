"use client";
import type { NavItem } from "@/components/navigation/sidebar/workspace-navigations";
import { trpc } from "@/lib/trpc/client";
import { useMemo } from "react";

/**
 * Hook to enhance Project resource navigation with actual project name.
 * Fetches the project list to find the project and updates the label.
 */
export const useProjectData = (baseNavItems: NavItem[], projectId?: string) => {
  // Fetch all projects
  const { data: projects, isLoading } = trpc.deploy.project.list.useQuery(undefined, {
    enabled: !!projectId,
  });

  const enhancedNavItems = useMemo(() => {
    if (!projectId) {
      return baseNavItems;
    }

    // If loading, mark the item as loading
    if (isLoading) {
      return baseNavItems.map((item) => {
        if (item.label === projectId) {
          return {
            ...item,
            loading: true,
          };
        }
        return item;
      });
    }

    if (!projects) {
      return baseNavItems;
    }

    // Find the project in the fetched data
    const project = projects.find((p) => p.id === projectId);

    // If project not found, return base items unchanged
    if (!project) {
      return baseNavItems;
    }

    // Update the parent project item label with the actual project name
    return baseNavItems.map((item) => {
      // Check if this is the project parent item
      if (item.label === projectId) {
        return {
          ...item,
          label: project.name,
          loading: false,
        };
      }
      return item;
    });
  }, [baseNavItems, projects, projectId, isLoading]);

  // Get the project name if available
  const projectName = useMemo(() => {
    if (!projects || !projectId) {
      return undefined;
    }
    const project = projects.find((p) => p.id === projectId);
    return project?.name;
  }, [projects, projectId]);

  return {
    enhancedNavItems,
    projectName,
  };
};
