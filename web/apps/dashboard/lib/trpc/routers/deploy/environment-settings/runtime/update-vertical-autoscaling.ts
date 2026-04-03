import { and, db, eq } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { appRegionalSettings, verticalAutoscalingPolicies } from "@unkey/db/src/schema";
import { newId } from "@unkey/id";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

export const updateVerticalAutoscaling = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      enabled: z.boolean(),
      updateMode: z.enum(["off", "initial", "recreate", "in_place_or_recreate"]).default("off"),
      controlledResources: z.enum(["cpu", "memory", "both"]).default("both"),
      controlledValues: z.enum(["requests", "requests_and_limits"]).default("requests"),
      cpuMinMillicores: z.number().int().positive().nullable().default(null),
      cpuMaxMillicores: z.number().int().positive().nullable().default(null),
      memoryMinMib: z.number().int().positive().nullable().default(null),
      memoryMaxMib: z.number().int().positive().nullable().default(null),
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

      if (!input.enabled) {
        // Disable VPA: remove policy reference from all regional settings
        const existingPolicyId = regions[0].verticalAutoscalingPolicyId;

        await tx
          .update(appRegionalSettings)
          .set({
            verticalAutoscalingPolicyId: null,
          })
          .where(
            and(
              eq(appRegionalSettings.workspaceId, ctx.workspace.id),
              eq(appRegionalSettings.environmentId, input.environmentId),
            ),
          );

        // Clean up the policy row if it exists
        if (existingPolicyId) {
          await tx
            .delete(verticalAutoscalingPolicies)
            .where(eq(verticalAutoscalingPolicies.id, existingPolicyId));
        }
        return;
      }

      // Enable VPA: create or update policy, set FK, clear HPA FK (mutual exclusion)
      const existingPolicyId = regions[0].verticalAutoscalingPolicyId;
      const policyId = existingPolicyId ?? newId("autoscalingPolicy");

      const policyValues = {
        updateMode: input.updateMode,
        controlledResources: input.controlledResources,
        controlledValues: input.controlledValues,
        cpuMinMillicores: input.cpuMinMillicores,
        cpuMaxMillicores: input.cpuMaxMillicores,
        memoryMinMib: input.memoryMinMib,
        memoryMaxMib: input.memoryMaxMib,
        updatedAt: Date.now(),
      };

      if (existingPolicyId) {
        await tx
          .update(verticalAutoscalingPolicies)
          .set(policyValues)
          .where(eq(verticalAutoscalingPolicies.id, existingPolicyId));
      } else {
        await tx.insert(verticalAutoscalingPolicies).values({
          id: policyId,
          workspaceId: ctx.workspace.id,
          ...policyValues,
          createdAt: Date.now(),
        });
      }

      // Set VPA FK and clear HPA FK (a workload uses either HPA or VPA, not both)
      await tx
        .update(appRegionalSettings)
        .set({
          verticalAutoscalingPolicyId: policyId,
          horizontalAutoscalingPolicyId: null,
        })
        .where(
          and(
            eq(appRegionalSettings.workspaceId, ctx.workspace.id),
            eq(appRegionalSettings.environmentId, input.environmentId),
          ),
        );
    });
  });
