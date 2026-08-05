import { and, db, eq } from "@/lib/db";
import { freeTierLimits } from "@/lib/quotas";
import { TRPCError } from "@trpc/server";
import { appRegionalSettings, horizontalAutoscalingPolicies, limits } from "@unkey/db/src/schema";
import { newId } from "@unkey/id";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

export const updateInstances = workspaceProcedure
  .input(
    z
      .object({
        environmentId: z.string(),
        replicasMin: z.number().int().min(1),
        replicasMax: z.number().int().min(1),
      })
      .refine((d) => d.replicasMin <= d.replicasMax, {
        message: "replicasMin must be ≤ replicasMax",
        path: ["replicasMin"],
      }),
  )
  .mutation(async ({ ctx, input }) => {
    const workspaceLimits = await db.query.limits.findFirst({
      where: eq(limits.workspaceId, ctx.workspace.id),
      columns: { autoscalingReplicasMax: true },
    });

    const maxPerRegion = Math.max(
      1,
      workspaceLimits?.autoscalingReplicasMax ?? freeTierLimits.autoscalingReplicasMax,
    );
    if (input.replicasMax > maxPerRegion) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: `Instances per region cannot exceed ${maxPerRegion}. Contact support@unkey.com to increase it.`,
      });
    }

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
            replicasMin: input.replicasMin,
            replicasMax: input.replicasMax,
            cpuThreshold: 80,
          })
          .where(eq(horizontalAutoscalingPolicies.id, existingPolicyId));
      } else {
        await tx.insert(horizontalAutoscalingPolicies).values({
          id: policyId,
          workspaceId: ctx.workspace.id,
          replicasMin: input.replicasMin,
          replicasMax: input.replicasMax,
          cpuThreshold: 80,
          createdAt: Date.now(),
        });
      }

      await tx
        .update(appRegionalSettings)
        .set({
          replicas: input.replicasMax,
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
