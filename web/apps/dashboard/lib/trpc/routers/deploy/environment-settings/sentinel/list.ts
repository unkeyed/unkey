import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";
import { loadOwnedEnvironment, loadPolicies } from "./_shared";

export const list = workspaceProcedure
  .input(z.object({ environmentId: z.string().min(1) }))
  .query(async ({ ctx, input }) => {
    await loadOwnedEnvironment(ctx.workspace.id, input.environmentId);
    const policies = await loadPolicies(ctx.workspace.id, input.environmentId);
    return { policies };
  });
