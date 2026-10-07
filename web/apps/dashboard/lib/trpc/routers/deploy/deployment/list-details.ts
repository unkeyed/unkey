import { and, db, eq, inArray } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { deployments } from "@unkey/db/src/schema";
import { z } from "zod";
import { type DeploymentDetails, loadDeploymentDetails } from "./enrich-deployment-rows";

// The deployments collection reads rows from the public API and merges in what
// the API does not return: parent ids, the desired state, the trigger reason,
// and the runtime details
export const listDeploymentDetails = workspaceProcedure
  .input(z.object({ deploymentIds: z.array(z.string()).min(1).max(100) }))
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx, input }) => {
    const rows = await db
      .select({
        id: deployments.id,
        appId: deployments.appId,
        environmentId: deployments.environmentId,
        desiredState: deployments.desiredState,
        triggerReason: deployments.triggerReason,
      })
      .from(deployments)
      .where(
        and(
          eq(deployments.workspaceId, ctx.workspace.id),
          inArray(deployments.id, input.deploymentIds),
        ),
      );
    const details = await loadDeploymentDetails(ctx.workspace.id, rows);

    const result: Record<string, (typeof rows)[number] & DeploymentDetails> = {};
    for (const row of rows) {
      const detail = details.get(row.id);
      if (detail) {
        result[row.id] = { ...row, ...detail };
      }
    }
    return result;
  });
