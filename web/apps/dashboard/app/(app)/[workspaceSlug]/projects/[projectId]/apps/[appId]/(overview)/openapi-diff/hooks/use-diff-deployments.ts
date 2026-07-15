"use client";
import { collection } from "@/lib/collections";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { useAppId, useProjectData } from "../../data-provider";
export const useDiffDeployments = () => {
  const { projectId } = useProjectData();
  const appId = useAppId();
  const deployments = useLiveQuery(
    (q) => {
      const environments = q
        .from({ environment: collection.environments })
        .where(({ environment }) =>
          and(eq(environment.projectId, projectId), eq(environment.appId, appId)),
        );
      return q
        .from({ deployment: collection.deployments })
        .where(({ deployment }) =>
          and(eq(deployment.projectId, projectId), eq(deployment.appId, appId)),
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
