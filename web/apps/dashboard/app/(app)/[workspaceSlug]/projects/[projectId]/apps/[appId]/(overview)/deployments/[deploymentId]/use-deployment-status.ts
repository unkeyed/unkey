import { isDeploymentInFlight } from "@/lib/collections/deploy/deployment-status";
import type { Deployment } from "@/lib/collections/deploy/deployments";
import { trpc } from "@/lib/trpc/client";
import { useMemo } from "react";
import { deriveStatusFromSteps } from "./deployment-utils";

const IN_FLIGHT_POLL_MS = 1_000;
// While the build runs, a finished build step in the logs triggers a refetch,
// see useBuildLogs. This poll catches a build that ends without a log line,
// such as a cancel, a timeout, or a collapsed log panel that does not poll
const BUILDING_POLL_MS = 5_000;

/**
 * Owns the deployment's step polling and the status derived from it. The
 * detail header (Cancel/Redeploy eligibility) and the overview page both read
 * from here rather than the raw collection status, which lags steps by ~1s.
 * React Query dedupes the poll by query key, so two callers share one request.
 */
export function useDeploymentStatus(deployment: Deployment) {
  const skipped = deployment.status === "skipped";

  const steps = trpc.deploy.deployment.steps.useQuery(
    { deploymentId: deployment.id },
    {
      refetchInterval: (data) => {
        if (!isDeploymentInFlight(deployment.status)) {
          return false;
        }
        return data?.building && data.building.endedAt === null
          ? BUILDING_POLL_MS
          : IN_FLIGHT_POLL_MS;
      },
      refetchOnWindowFocus: false,
      enabled: !skipped && deployment.status !== "superseded" && deployment.status !== "cancelled",
    },
  );

  const derivedStatus = useMemo(
    () => (skipped ? ("skipped" as const) : deriveStatusFromSteps(steps.data, deployment.status)),
    [steps.data, deployment.status, skipped],
  );

  return { steps, derivedStatus };
}
