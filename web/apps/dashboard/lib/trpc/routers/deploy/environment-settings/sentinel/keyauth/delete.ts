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
    const [env, current] = await Promise.all([
      loadOwnedEnvironment(ctx.workspace.id, input.environmentId),
      loadPolicies(ctx.workspace.id, input.environmentId),
    ]);
    const idx = current.findIndex((p) => p.id === input.policyId);
    if (idx === -1) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `Policy ${input.policyId} not found`,
      });
    }
    if (current[idx].type !== "keyauth") {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: `Policy ${input.policyId} is not a keyauth policy`,
      });
    }

    const next = current.filter((p) => p.id !== input.policyId);
    await savePolicies(ctx.workspace.id, input.environmentId, env.appId, next);
    return { policyId: input.policyId };
  });
