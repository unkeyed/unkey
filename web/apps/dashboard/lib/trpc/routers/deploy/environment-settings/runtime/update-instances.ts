import { and, db, eq } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { appRegionalSettings, horizontalAutoscalingPolicies } from "@unkey/db/src/schema";
import { newId } from "@unkey/id";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

export const updateInstances = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      replicasPerRegion: z
        .number()
        .min(1)
        .max(
          4,
          "Instances are limited to 4 per region during beta. Please contact support@unkey.com if you need more.",
        ),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    await db.transaction(async (tx) => {
      const regions = await tx.query.appRegionalSettings.findMany({
        where: and(
          eq(appRegionalSettings.workspaceId, ctx.workspace.id),
          eq(appRegionalSettings.environmentId, input.environmentId),
        ),
      });

      if (regions.length === 0) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "No regional settings found for this environment.",
        });
      }

      const existingPolicyId = regions[0].horizontalAutoscalingPolicyId;
      const policyId = existingPolicyId ?? newId("autoscalingPolicy");

      if (existingPolicyId) {
        await tx
          .update(horizontalAutoscalingPolicies)
          .set({
            replicasMin: 1,
            replicasMax: input.replicasPerRegion,
            cpuThreshold: 80,
          })
          .where(eq(horizontalAutoscalingPolicies.id, existingPolicyId));
      } else {
        await tx.insert(horizontalAutoscalingPolicies).values({
          id: policyId,
          workspaceId: ctx.workspace.id,
          replicasMin: 1,
          replicasMax: input.replicasPerRegion,
          cpuThreshold: 80,
          createdAt: Date.now(),
        });
      }

      await tx
        .update(appRegionalSettings)
        .set({
          replicas: input.replicasPerRegion,
          horizontalAutoscalingPolicyId: policyId,
        })
        .where(
          and(
            eq(appRegionalSettings.workspaceId, ctx.workspace.id),
            eq(appRegionalSettings.environmentId, input.environmentId),
          ),
        );
    });
  });
