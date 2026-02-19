"use client";

import { collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { useLiveQuery } from "@tanstack/react-db";
import { useMemo } from "react";

/**
 * Hook to fetch resource name by ID and type
 */
export function useResourceName(
  resourceType?: "api" | "project" | "namespace",
  resourceId?: string,
) {
  // Fetch APIs
  const { data: apiData } = trpc.api.overview.query.useInfiniteQuery(
    { limit: 18 },
    {
      enabled: resourceType === "api" && !!resourceId,
      getNextPageParam: (lastPage) => lastPage.nextCursor,
    },
  );

  // Fetch Projects using LiveQuery
  const { data: projectData } = useLiveQuery((q) => q.from({ project: collection.projects }));

  // Fetch Namespaces (ratelimits)
  const { data: namespaceData } = trpc.ratelimit.namespace.list.useQuery(undefined, {
    enabled: resourceType === "namespace" && !!resourceId,
  });

  const resourceName = useMemo(() => {
    if (!resourceType || !resourceId) {
      return undefined;
    }

    switch (resourceType) {
      case "api": {
        if (!apiData?.pages) {
          return undefined;
        }
        const api = apiData.pages.flatMap((page) => page.apiList).find((a) => a.id === resourceId);
        return api?.name;
      }
      case "project": {
        if (!projectData) {
          return undefined;
        }
        const project = projectData.find((p) => p.id === resourceId);
        return project?.name;
      }
      case "namespace": {
        if (!namespaceData) {
          return undefined;
        }
        const namespace = namespaceData.find((n) => n.id === resourceId);
        return namespace?.name;
      }
    }
  }, [resourceType, resourceId, apiData?.pages, projectData, namespaceData]);

  return resourceName;
}
