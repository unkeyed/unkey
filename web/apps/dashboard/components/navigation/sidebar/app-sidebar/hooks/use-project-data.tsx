"use client";
import type { NavItem } from "@/components/navigation/sidebar/workspace-navigations";
import { trpc } from "@/lib/trpc/client";
import { useMemo } from "react";

/**
 * Hook to enhance Project resource navigation with actual project name.
 * Fetches the project list to find the project and updates the label.
 */
export const useProjectData = (baseNavItems: NavItem[], projectSlug?: string) => {
  // Fetch all projects
  const { data: projects, isLoading } = trpc.deploy.project.list.useQuery(undefined, {
    enabled: !!projectSlug,
  });

  const enhancedNavItems = useMemo(() => {
    if (!projectSlug) {
      return baseNavItems;
    }

    // If loading, mark the item as loading
    if (isLoading) {
      return baseNavItems.map((item) => {
        if (item.label === projectSlug) {
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
    const project = projects.find((p) => p.slug === projectSlug);

    // If project not found, return base items unchanged
    if (!project) {
      return baseNavItems;
    }

    // Update the parent project item label with the actual project name
    return baseNavItems.map((item) => {
      // Check if this is the project parent item
      if (item.label === projectSlug) {
        return {
          ...item,
          label: project.name,
          loading: false,
        };
      }
      return item;
    });
  }, [baseNavItems, projects, projectSlug, isLoading]);

  // Get the project name if available
  const projectName = useMemo(() => {
    if (!projects || !projectSlug) {
      return undefined;
    }
    const project = projects.find((p) => p.slug === projectSlug);
    return project?.name;
  }, [projects, projectSlug]);

  return {
    enhancedNavItems,
    projectName,
  };
};
