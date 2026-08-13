import type { Policy } from "@/lib/collections/deploy/policies.schema";
import { db } from "@/lib/db";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";
import { loadOwnedEnvironment, loadPolicies, savePolicies } from "./_shared";

export const reorder = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string().min(1),
      policyIds: z.array(z.string().min(1)),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const env = await loadOwnedEnvironment(ctx.workspace.id, input.environmentId);

    const reorderedIds = await db.transaction(async (tx) => {
      const current = await loadPolicies(ctx.workspace.id, input.environmentId, tx);
      const reordered = reconcileOrder(current, input.policyIds);
      await savePolicies(ctx.workspace.id, input.environmentId, env.appId, reordered, tx);
      return reordered.map((p) => p.id);
    });
    return { policyIds: reorderedIds };
  });

/**
 * Unknown ids in the input are dropped; current ids
 * the client didn't mention (e.g. a teammate just added one) keep their
 * original positions appended at the end.
 * */
function reconcileOrder(current: Policy[], requestedIds: string[]): Policy[] {
  const byId = new Map(current.map((p) => [p.id, p]));

  const fromClient = Array.from(new Set(requestedIds))
    .map((id) => byId.get(id))
    .filter((p): p is Policy => p !== undefined);

  const placed = new Set(fromClient.map((p) => p.id));
  const remaining = current.filter((p) => !placed.has(p.id));

  return [...fromClient, ...remaining];
}
