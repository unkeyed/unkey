import type { SentinelPolicy } from "@/lib/collections/deploy/sentinel-policies.schema";
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
    const [env, current] = await Promise.all([
      loadOwnedEnvironment(ctx.workspace.id, input.environmentId),
      loadPolicies(ctx.workspace.id, input.environmentId),
    ]);
    const reordered = reconcileOrder(current, input.policyIds);

    await savePolicies(ctx.workspace.id, input.environmentId, env.appId, reordered);
    return { policyIds: reordered.map((p) => p.id) };
  });

/**
 * Unknown ids in the input are dropped; current ids
 * the client didn't mention (e.g. a teammate just added one) keep their
 * original positions appended at the end.
 * */
function reconcileOrder(current: SentinelPolicy[], requestedIds: string[]): SentinelPolicy[] {
  const byId = new Map(current.map((p) => [p.id, p]));

  const fromClient = Array.from(new Set(requestedIds))
    .map((id) => byId.get(id))
    .filter((p): p is SentinelPolicy => p !== undefined);

  const placed = new Set(fromClient.map((p) => p.id));
  const remaining = current.filter((p) => !placed.has(p.id));

  return [...fromClient, ...remaining];
}
