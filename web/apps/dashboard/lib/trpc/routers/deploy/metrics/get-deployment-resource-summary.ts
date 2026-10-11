import { clickhouse } from "@/lib/clickhouse";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireOwnedDeployment } from "./require-owned-deployment";

export const getDeploymentResourceSummary = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      resourceId: z.string(),
      instanceName: z.string().optional(),
    }),
  )
  .query(async ({ ctx, input }) => {
    await requireOwnedDeployment(
      ctx.deploymentOwnership,
      ctx.workspace.id,
      input.resourceId,
      "Resource not found",
    );

    const result = await clickhouse.resources.summary({
      workspaceId: ctx.workspace.id,
      resourceId: input.resourceId,
      instanceName: input.instanceName ?? "",
    });

    if (result.err) {
      // Bubble a typed error so the UI can render a degraded state
      // instead of treating a ClickHouse outage as "no data". The
      // resource-existence check above already returns NOT_FOUND for
      // truly missing resources, so reaching here means a real failure.
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch resource summary",
        cause: result.err,
      });
    }

    return result.val[0] ?? null;
  });
