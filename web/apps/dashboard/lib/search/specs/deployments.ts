import {
  GROUPED_DEPLOYMENT_STATUSES,
  deploymentListFilterFieldConfig,
  deploymentListFilterOutputSchema,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/deployments/filters.schema";
import type { SearchSpec } from "../spec";

export const deploymentsSearchSpec: SearchSpec = {
  subject: "deployments",
  outputSchema: deploymentListFilterOutputSchema,
  config: deploymentListFilterFieldConfig,
  durationField: "since",
  fields: {
    status: "where the deployment got to",
    environment: "the environment it went to, such as production or preview",
    branch: "the git branch it was built from",
    since: "how far back the search reaches, as a lookback from now",
    startTime: "the start of a named range, as a millisecond timestamp",
    endTime: "the end of a named range, as a millisecond timestamp",
  },
  rules: [
    `Status is one of: ${GROUPED_DEPLOYMENT_STATUSES.join(", ")}.`,
    "blocked means waiting for someone to approve it. superseded means a newer deployment took over. stopped means it was shut down.",
  ],
  examples: [
    {
      query: "failed deployments in the last 24h",
      result: [
        { field: "status", filters: [{ operator: "is", value: "failed" }] },
        { field: "since", filters: [{ operator: "is", value: "24h" }] },
      ],
    },
    {
      query: "deployments on main",
      result: [{ field: "branch", filters: [{ operator: "is", value: "main" }] }],
    },
    {
      query: "production deployments that are still building",
      result: [
        { field: "status", filters: [{ operator: "is", value: "building" }] },
        { field: "environment", filters: [{ operator: "is", value: "production" }] },
      ],
    },
    {
      query: "deployments waiting for approval",
      result: [{ field: "status", filters: [{ operator: "is", value: "blocked" }] }],
    },
    {
      query: "show me stopped deployments",
      result: [{ field: "status", filters: [{ operator: "is", value: "stopped" }] }],
    },
    { query: "list deployments", result: [], note: "nothing is named, so nothing is filtered" },
  ],
};
