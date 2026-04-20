import { and, db, eq } from "@/lib/db";
import { workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { sentinels } from "@unkey/db/src/schema";
import { z } from "zod";
import { getCtrlClients } from "../../ctrl";

// changeTier delegates to the ctrl service, which does the DB transaction
// (new subscription + repoint + deploy_status=progressing + outbox) and
// enqueues the Restate Deploy workflow. We only do the workspace-scoped
// ownership check here so we don't leak sentinel IDs across workspaces.
export const changeTier = workspaceProcedure
  .input(
    z.object({
      sentinelId: z.string(),
      tierId: z.string(),
      tierVersion: z.string(),
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
    await ctrl.sentinel.changeTier({
      sentinelId: input.sentinelId,
      tierId: input.tierId,
      tierVersion: input.tierVersion,
    });
    return {};
  });
