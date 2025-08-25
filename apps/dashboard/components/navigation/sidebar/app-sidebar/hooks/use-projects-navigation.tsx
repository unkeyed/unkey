"use client";
import type { NavItem } from "@/components/navigation/sidebar/workspace-navigations";
import { trpc } from "@/lib/trpc/client";
import { useSelectedLayoutSegments } from "next/navigation";
import { useMemo } from "react";

export const useProjectNavigation = (baseNavItems: NavItem[]) => {
  const segments = useSelectedLayoutSegments() ?? [];
  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading } =
    trpc.deploy.project.list.useInfiniteQuery(
      { query: [] },
      {
        getNextPageParam: (lastPage) => lastPage.nextCursor,
      },
    );

  const projectNavItems = useMemo(() => {
    if (!data?.pages) {
      return [];
    }
    return data.pages.flatMap((page) =>
      page.projects.map((project) => {
        const currentProjectActive = segments.at(0) === "projects" && segments.at(1) === project.id;

        const projectNavItem: NavItem = {
          href: `/projects/${project.id}`,
          icon: null,
          label: project.name,
          active: currentProjectActive,
          showSubItems: true,
        };

        return projectNavItem;
      }),
    );
  }, [data?.pages, segments]);

  const enhancedNavItems = useMemo(() => {
    const items = [...baseNavItems];
    const projectsItemIndex = items.findIndex((item) => item.href === "/projects");

    if (projectsItemIndex !== -1) {
      const projectsItem = { ...items[projectsItemIndex] };
      projectsItem.showSubItems = true;
      projectsItem.items = [...(projectsItem.items || []), ...projectNavItems];

      if (hasNextPage) {
        projectsItem.items?.push({
          icon: () => null,
          href: "#load-more-projects",
          label: <div className="font-normal decoration-dotted underline">More</div>,
          active: false,
          loadMoreAction: true,
        });
      }

      items[projectsItemIndex] = projectsItem;
    }

    return items;
  }, [baseNavItems, projectNavItems, hasNextPage]);

  const loadMore = () => {
    if (!isFetchingNextPage && hasNextPage) {
      fetchNextPage();
    }
  };

  return {
    enhancedNavItems,
    isLoading,
    loadMore,
  };
};
