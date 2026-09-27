import { isDeploymentInFlight } from "@/lib/collections/deploy/deployment-status";
import type { Deployment } from "@/lib/collections/deploy/deployments";
import { trpc } from "@/lib/trpc/client";
import { useMemo } from "react";
import { deriveStatusFromSteps } from "./deployment-utils";

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
      refetchInterval: isDeploymentInFlight(deployment.status) ? 1_000 : false,
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
