import { isDeploymentInFlight } from "@/lib/collections/deploy/deployment-status";
import type { Deployment } from "@/lib/collections/deploy/deployments";
import { trpc } from "@/lib/trpc/client";

const IN_FLIGHT_POLL_MS = 1_000;
const SETTLED_POLL_MS = 10_000;
// ctrl flushes build logs to ClickHouse up to 2s after they are written, so
// the last lines can arrive after the deployment settles. The window is far
// longer than the flush interval to cover slow or retried inserts.
const SETTLED_POLL_WINDOW_MS = 5 * 60_000;

export function useBuildSteps(deployment: Deployment) {
  const inFlight = isDeploymentInFlight(deployment.status);
  const settledAt = deployment.updatedAt ?? deployment.createdAt;

  return trpc.deploy.deployment.buildSteps.useQuery(
    { deploymentId: deployment.id, includeStepLogs: true },
    {
      refetchInterval: () => {
        if (inFlight) {
          return IN_FLIGHT_POLL_MS;
        }
        return Date.now() - settledAt < SETTLED_POLL_WINDOW_MS ? SETTLED_POLL_MS : false;
      },
    },
  );
}
