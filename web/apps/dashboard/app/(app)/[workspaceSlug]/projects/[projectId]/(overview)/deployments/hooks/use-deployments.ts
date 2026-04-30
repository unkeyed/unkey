import { collection } from "@/lib/collections";
import { DEPLOYMENTS_DEFAULT_LIMIT } from "@/lib/collections/deploy/deployments";
import type { Environment } from "@/lib/collections/deploy/environments";
import { parseDuration } from "@/lib/duration";
import { eq, gte, lte, useLiveQuery } from "@tanstack/react-db";
import { useMemo } from "react";
import { useProjectData } from "../../data-provider";
import type { DeploymentListFilterField } from "../filters.schema";
import { useFilters } from "./use-filters";

export const useDeployments = () => {
  const { projectId, environments } = useProjectData();
  const { filters } = useFilters();

  const environmentMap = useMemo(() => {
    const map = new Map<string, Environment>();
    for (const env of environments) {
      map.set(env.id, env);
    }
    return map;
  }, [environments]);

  const startTime = filters.find((f) => f.field === "startTime")?.value as number | undefined;
  const endTime = filters.find((f) => f.field === "endTime")?.value as number | undefined;
  const since = filters.find((f) => f.field === "since")?.value as string | undefined;
  const sinceMs = useMemo(() => (since ? Date.now() - parseDuration(since) : undefined), [since]);

  const result = useLiveQuery(
    (q) => {
      let query = q
        .from({ deployment: collection.deployments })
        .where(({ deployment }) => eq(deployment.projectId, projectId));

      if (startTime !== undefined) {
        query = query.where(({ deployment }) => gte(deployment.createdAt, startTime));
      }
      if (endTime !== undefined) {
        query = query.where(({ deployment }) => lte(deployment.createdAt, endTime));
      }
      if (sinceMs !== undefined) {
        query = query.where(({ deployment }) => gte(deployment.createdAt, sinceMs));
      }

      return query
        .orderBy(({ deployment }) => deployment.createdAt, "desc")
        .limit(DEPLOYMENTS_DEFAULT_LIMIT);
    },
    [projectId, startTime, endTime, sinceMs],
  );

  const deployments = useMemo(() => {
    const withEnvironments = result.data.map((deployment) => ({
      deployment,
      environment: environmentMap.get(deployment.environmentId),
    }));

    const clientFilters = filters.filter(
      (f) => f.field !== "startTime" && f.field !== "endTime" && f.field !== "since",
    );

    if (clientFilters.length === 0) {
      return { isLoading: result.isLoading, data: withEnvironments };
    }

    const groupedFilters = clientFilters.reduce(
      (acc, f) => {
        if (!acc[f.field]) {
          acc[f.field] = [];
        }
        acc[f.field].push(f.value);
        return acc;
      },
      {} as Record<DeploymentListFilterField, (string | number)[]>,
    );

    const filtered = withEnvironments.filter(({ deployment, environment }) => {
      for (const [field, values] of Object.entries(groupedFilters)) {
        const f = field as DeploymentListFilterField;
        switch (f) {
          case "status":
            if (!values.includes(deployment.status)) {
              return false;
            }
            break;
          case "branch":
            if (!values.includes(deployment.gitBranch)) {
              return false;
            }
            break;
          case "environment":
            if (!environment || !values.includes(environment.slug)) {
              return false;
            }
            break;
        }
      }

      return true;
    });

    return { isLoading: result.isLoading, data: filtered };
  }, [result, filters, environmentMap]);

  return {
    deployments,
  };
};
