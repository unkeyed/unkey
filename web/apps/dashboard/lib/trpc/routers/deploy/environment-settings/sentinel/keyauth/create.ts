import { keyauthPolicySchema } from "@/lib/collections/deploy/sentinel-policies.schema";
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
    const env = await loadOwnedEnvironment(ctx.workspace.id, input.environmentId);
    await assertKeyspacesOwned(ctx.workspace.id, input.policy.keyauth.keySpaceIds);

    const current = await loadPolicies(ctx.workspace.id, input.environmentId);
    if (current.some((p) => p.id === input.policy.id)) {
      throw new TRPCError({
        code: "CONFLICT",
        message: `Policy ${input.policy.id} already exists`,
      });
    }

    const next = [...current, input.policy];
    await savePolicies(ctx.workspace.id, input.environmentId, env.appId, next);
    return { policy: input.policy };
  });
