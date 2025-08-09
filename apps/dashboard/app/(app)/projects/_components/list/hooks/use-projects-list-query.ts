import { trpc } from "@/lib/trpc/client";
import type { Project } from "@/lib/trpc/routers/deploy/project/list";
import { useEffect, useMemo, useState } from "react";
import { useProjectsFilters } from "../../hooks/use-projects-filters";
import {
  projectsFilterFieldConfig,
  projectsListFilterFieldNames,
} from "../../projects-filters.schema";
import type { ProjectsQueryPayload } from "../projects-list.schema";

export function useProjectsListQuery() {
  const [totalCount, setTotalCount] = useState(0);
  const [projectsMap, setProjectsMap] = useState(() => new Map<string, Project>());
  const { filters } = useProjectsFilters();

  const projects = useMemo(() => Array.from(projectsMap.values()), [projectsMap]);

  const queryParams = useMemo(() => {
    const params: ProjectsQueryPayload = {
      ...Object.fromEntries(projectsListFilterFieldNames.map((field) => [field, []])),
    };

    filters.forEach((filter) => {
      if (!projectsListFilterFieldNames.includes(filter.field) || !params[filter.field]) {
        return;
      }

      const fieldConfig = projectsFilterFieldConfig[filter.field];
      const validOperators = fieldConfig.operators;

      if (!validOperators.includes(filter.operator)) {
        throw new Error("Invalid operator");
      }

      if (typeof filter.value === "string") {
        params[filter.field]?.push({
          operator: filter.operator,
          value: filter.value,
        });
      }
    });

    return params;
  }, [filters]);

  const {
    data: projectData,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    isLoading: isLoadingInitial,
  } = trpc.deploy.project.list.useInfiniteQuery(queryParams, {
    getNextPageParam: (lastPage) => lastPage.nextCursor,
    staleTime: Number.POSITIVE_INFINITY,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });

  useEffect(() => {
    if (projectData) {
      const newMap = new Map<string, Project>();
      projectData.pages.forEach((page) => {
        page.projects.forEach((project) => {
          newMap.set(project.id, project);
        });
      });

      if (projectData.pages.length > 0) {
        setTotalCount(projectData.pages[0].total);
      }

      setProjectsMap(newMap);
    }
  }, [projectData]);

  return {
    projects,
    isLoading: isLoadingInitial,
    hasMore: hasNextPage,
    loadMore: fetchNextPage,
    isLoadingMore: isFetchingNextPage,
    totalCount,
  };
}
