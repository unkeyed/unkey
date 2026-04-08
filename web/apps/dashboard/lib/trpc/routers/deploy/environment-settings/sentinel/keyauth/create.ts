import { keyauthPolicySchema } from "@/lib/collections/deploy/sentinel-policies.schema";
import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../../../../trpc";
import { assertKeyspacesOwned, loadOwnedEnvironment, loadPolicies, savePolicies } from "../_shared";

export const create = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string().min(1),
      policy: keyauthPolicySchema,
    }),
  )
  .mutation(async ({ ctx, input }) => {
    // Workspace-scoped ownership check is safe outside the txn (read-only).
    await assertKeyspacesOwned(ctx.workspace.id, input.policy.keyauth.keySpaceIds);

    // RMW on the policies blob must be atomic — load + save in one transaction
    // so concurrent edits from another tab can't be silently dropped.
    await db.transaction(async (tx) => {
      const env = await loadOwnedEnvironment(ctx.workspace.id, input.environmentId, tx);
      const current = await loadPolicies(ctx.workspace.id, input.environmentId, tx);
      if (current.some((p) => p.id === input.policy.id)) {
        throw new TRPCError({
          code: "CONFLICT",
          message: `Policy ${input.policy.id} already exists`,
        });
      }

      const next = [...current, input.policy];
      await savePolicies(ctx.workspace.id, input.environmentId, env.appId, next, tx);
    });
    return { policy: input.policy };
  });
