import { collection } from "@/lib/collections";
import { eq, gt, gte, lte, or, useLiveQuery } from "@tanstack/react-db";
import ms from "ms";
import { useProjectLayout } from "../../layout-provider";
import type { DeploymentListFilterField } from "../filters.schema";
import { useFilters } from "./use-filters";

export const useDeployments = () => {
  const { projectId, collections } = useProjectLayout();
  const { filters } = useFilters();

  const project = useLiveQuery((q) => {
    return q
      .from({ project: collection.projects })
      .where(({ project }) => eq(project.id, projectId))
      .orderBy(({ project }) => project.id, "asc")
      .limit(1);
  }).data.at(0);
  const liveDeploymentId = project?.liveDeploymentId;
  const liveDeployment = useLiveQuery(
    (q) =>
      q
        .from({ deployment: collections.deployments })
        .where(({ deployment }) => eq(deployment.id, liveDeploymentId))
        .orderBy(({ deployment }) => deployment.createdAt, "desc")
        .limit(1),
    [liveDeploymentId]
  ).data.at(0);
  const deployments = useLiveQuery(
    (q) => {
      // Query filtered environments
      // further down below we use this to rightJoin with deployments to filter deployments by environment
      let environments = q.from({ environment: collections.environments });

      for (const filter of filters) {
        if (filter.field === "environment") {
          environments = environments.where(({ environment }) =>
            eq(environment.slug, filter.value)
          );
        }
      }

      let query = q
        .from({ deployment: collections.deployments })

        .where(({ deployment }) => eq(deployment.projectId, projectId));

      // add additional where clauses based on filters.
      // All of these are a locical AND

      const groupedFilters = filters.reduce((acc, f) => {
        if (!acc[f.field]) {
          acc[f.field] = [];
        }
        acc[f.field].push(f.value);
        return acc;
      }, {} as Record<DeploymentListFilterField, (string | number)[]>);
      for (const [field, values] of Object.entries(groupedFilters)) {
        // this is kind of dumb, but `or`s type doesn't allow spreaded args without
        // specifying the first two
        const [v1, v2, ...rest] = values;
        const f = field as DeploymentListFilterField; // I want some typesafety
        switch (f) {
          case "status":
            query = query.where(({ deployment }) =>
              or(
                eq(deployment.status, v1),
                eq(deployment.status, v2),
                ...rest.map((value) => eq(deployment.status, value))
              )
            );
            break;
          case "branch":
            query = query.where(({ deployment }) =>
              or(
                eq(deployment.gitBranch, v1),
                eq(deployment.gitBranch, v2),
                ...rest.map((value) => eq(deployment.gitBranch, value))
              )
            );
            break;
          case "environment":
            // We already filtered
            break;
          case "since":
            query = query.where(({ deployment }) =>
              gt(deployment.createdAt, Date.now() - ms(values.at(0) as string))
            );

            break;
          case "startTime":
            query = query.where(({ deployment }) =>
              gte(deployment.createdAt, values.at(0))
            );
            break;
          case "endTime":
            query = query.where(({ deployment }) =>
              lte(deployment.createdAt, values.at(0))
            );
            break;
          default:
            break;
        }
      }

      return query
        .rightJoin(
          { environment: environments },
          ({ environment, deployment }) =>
            eq(environment.id, deployment.environmentId)
        )
        .orderBy(({ deployment }) => deployment.createdAt, "desc")
        .limit(100);
    },
    [projectId, filters]
  );

  return {
    deployments,
    liveDeployment,
  };
};
