"use client";
import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useProjectData } from "../../data-provider";
export const useDiffDeployments = () => {
  const { projectId } = useProjectData();
  const deployments = useLiveQuery(
    (q) => {
      const environments = q
        .from({ environment: collection.environments })
        .where(({ environment }) => eq(environment.projectId, projectId));
      return q
        .from({ deployment: collection.deployments })
        .where(({ deployment }) => eq(deployment.projectId, projectId))
        .rightJoin({ environment: environments }, ({ environment, deployment }) =>
          eq(environment.id, deployment?.environmentId ?? ""),
        )
        .orderBy(({ deployment }) => deployment?.createdAt ?? 0, "desc")
        .limit(100);
    },
    [projectId],
  );
  const data = (deployments.data ?? []).flatMap((d) =>
    d.deployment ? [{ deployment: d.deployment, environment: d.environment }] : [],
  );
  return { deployments: data, isLoading: deployments.isLoading };
};
