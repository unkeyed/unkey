import { collection } from "@/lib/collections";
import { DEPLOYMENTS_DEFAULT_LIMIT } from "@/lib/collections/deploy/deployments";
import type { Environment } from "@/lib/collections/deploy/environments";
import { parseDuration } from "@/lib/duration";
import { and, eq, gte, lte, useLiveQuery } from "@tanstack/react-db";
import { useMemo } from "react";
import { useProjectData } from "../../data-provider";
import type { DeploymentListFilterField } from "../filters.schema";
import { useFilters } from "./use-filters";

export const useDeployments = () => {
  const { projectId, appId, environments } = useProjectData();
  const { filters } = useFilters();

  const environmentMap = useMemo(() => {
    const map = new Map<string, Environment>();
    for (const env of environments) {
      map.set(env.id, env);
    }
    return map;
  }, [environments]);

  const startTimeRaw = filters.find((f) => f.field === "startTime")?.value;
  const startTime = typeof startTimeRaw === "number" ? startTimeRaw : undefined;
  const endTimeRaw = filters.find((f) => f.field === "endTime")?.value;
  const endTime = typeof endTimeRaw === "number" ? endTimeRaw : undefined;
  const sinceRaw = filters.find((f) => f.field === "since")?.value;
  const since = typeof sinceRaw === "string" ? sinceRaw : undefined;
  const sinceMs = useMemo(() => (since ? Date.now() - parseDuration(since) : undefined), [since]);

  const result = useLiveQuery(
    (q) => {
      let query = q
        .from({ deployment: collection.deployments })
        .where(({ deployment }) =>
          appId
            ? and(eq(deployment.projectId, projectId), eq(deployment.appId, appId))
            : eq(deployment.projectId, projectId),
        );

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
    [projectId, appId, startTime, endTime, sinceMs],
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

    const groupedFilters = new Map<DeploymentListFilterField, (string | number)[]>();
    for (const f of clientFilters) {
      const existing = groupedFilters.get(f.field);
      if (existing) {
        existing.push(f.value);
      } else {
        groupedFilters.set(f.field, [f.value]);
      }
    }

    const filtered = withEnvironments.filter(({ deployment, environment }) => {
      for (const [field, values] of groupedFilters) {
        switch (field) {
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
