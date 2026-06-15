import type { Deployment } from "@/lib/collections/deploy/deployments";
import { trpc } from "@/lib/trpc/client";
import { useMemo } from "react";
import { deriveStatusFromSteps } from "./deployment-utils";

// Steps stop changing once the deployment authorizes (awaiting_approval) or
// reaches a terminal state, so polling pauses there.
const STABLE_STATUSES = ["ready", "skipped", "superseded", "cancelled", "awaiting_approval"];

/**
 * Owns the deployment's step polling and the status derived from it. The
 * detail header (Cancel/Redeploy eligibility) and the overview page both read
 * from here rather than the raw collection status, which lags steps by ~1s.
 * React Query dedupes the poll by query key, so two callers share one request.
 */
export function useDeploymentStatus(deployment: Deployment) {
  const skipped = deployment.status === "skipped";
  const stepsAreStable = STABLE_STATUSES.includes(deployment.status);

  const steps = trpc.deploy.deployment.steps.useQuery(
    { deploymentId: deployment.id },
    {
      refetchInterval: stepsAreStable ? false : 1_000,
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
