"use client";
import { collection } from "@/lib/collections";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { useProjectData } from "../../data-provider";
export const useDiffDeployments = () => {
  const { projectId, appId } = useProjectData();
  const deployments = useLiveQuery(
    (q) => {
      const environments = q
        .from({ environment: collection.environments })
        .where(({ environment }) =>
          appId
            ? and(eq(environment.projectId, projectId), eq(environment.appId, appId))
            : eq(environment.projectId, projectId),
        );
      return q
        .from({ deployment: collection.deployments })
        .where(({ deployment }) =>
          appId
            ? and(eq(deployment.projectId, projectId), eq(deployment.appId, appId))
            : eq(deployment.projectId, projectId),
        )
        .rightJoin({ environment: environments }, ({ environment, deployment }) =>
          eq(environment.id, deployment?.environmentId ?? ""),
        )
        .orderBy(({ deployment }) => deployment?.createdAt ?? 0, "desc")
        .limit(100);
    },
    [projectId, appId],
  );
  const data = (deployments.data ?? []).flatMap((d) =>
    d.deployment ? [{ deployment: d.deployment, environment: d.environment }] : [],
  );
  return { deployments: data, isLoading: deployments.isLoading };
};
