import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../../../../trpc";
import { loadOwnedEnvironment, loadPolicies, savePolicies } from "../_shared";

export const remove = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string().min(1),
      policyId: z.string().min(1),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const env = await loadOwnedEnvironment(ctx.workspace.id, input.environmentId);

    await db.transaction(async (tx) => {
      const current = await loadPolicies(ctx.workspace.id, input.environmentId, tx);
      const idx = current.findIndex((p) => p.id === input.policyId);
      if (idx === -1) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: `Policy ${input.policyId} not found`,
        });
      }
      if (current[idx].type !== "firewall") {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: `Policy ${input.policyId} is not a firewall policy`,
        });
      }

      const next = current.filter((p) => p.id !== input.policyId);
      await savePolicies(ctx.workspace.id, input.environmentId, env.appId, next, tx);
    });
    return { policyId: input.policyId };
  });
