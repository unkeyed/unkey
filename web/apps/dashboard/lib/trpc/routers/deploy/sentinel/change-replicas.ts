import { and, db, eq } from "@/lib/db";
import { workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { sentinels } from "@unkey/db/src/schema";
import { z } from "zod";
import { getCtrlClients } from "../../ctrl";

// changeReplicas delegates to the ctrl service, which invokes the Restate
// Deploy workflow. Deploy owns the actual DB write + outbox + rollout wait,
// so we only do the workspace-scoped ownership check here.
export const changeReplicas = workspaceProcedure
  .input(
    z.object({
      sentinelId: z.string(),
      desiredReplicas: z.number().int().min(1).max(24),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const sentinel = await db.query.sentinels.findFirst({
      where: and(eq(sentinels.id, input.sentinelId), eq(sentinels.workspaceId, ctx.workspace.id)),
      columns: { id: true },
    });
    if (!sentinel) {
      throw new TRPCError({ code: "NOT_FOUND", message: "Sentinel not found" });
    }

    const ctrl = getCtrlClients();
    await ctrl.sentinel.changeReplicas({
      sentinelId: input.sentinelId,
      desiredReplicas: input.desiredReplicas,
    });
    return {};
  });
