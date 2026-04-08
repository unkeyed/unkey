import { keyauthPolicySchema } from "@/lib/collections/deploy/sentinel-policies.schema";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../../../../trpc";
import { assertKeyspacesOwned, loadOwnedEnvironment, loadPolicies, savePolicies } from "../_shared";

export const update = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string().min(1),
      policy: keyauthPolicySchema,
    }),
  )
  .mutation(async ({ ctx, input }) => {
    // So if keyspaces are not owned this will prevent loading the policies
    const [env, _, current] = await Promise.all([
      loadOwnedEnvironment(ctx.workspace.id, input.environmentId),
      assertKeyspacesOwned(ctx.workspace.id, input.policy.keyauth.keySpaceIds),
      loadPolicies(ctx.workspace.id, input.environmentId),
    ]);
    const idx = current.findIndex((p) => p.id === input.policy.id);
    if (idx === -1) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `Policy ${input.policy.id} not found`,
      });
    }
    if (current[idx].type !== "keyauth") {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: `Policy ${input.policy.id} is not a keyauth policy`,
      });
    }

    const next = [...current];
    next[idx] = input.policy;
    await savePolicies(ctx.workspace.id, input.environmentId, env.appId, next);
    return { policy: input.policy };
  });
