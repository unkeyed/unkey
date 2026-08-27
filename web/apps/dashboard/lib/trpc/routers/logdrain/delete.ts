import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

export const deleteLogdrain = workspaceProcedure
  .input(z.object({ id: z.string().min(1) }))
  .mutation(async ({ ctx, input }) => {
    try {
      await db.transaction(async (tx) => {
        const [drain] = await tx
          .select({ id: schema.logdrains.id, name: schema.logdrains.name })
          .from(schema.logdrains)
          .where(
            and(
              eq(schema.logdrains.id, input.id),
              eq(schema.logdrains.workspaceId, ctx.workspace.id),
            ),
          )
          .for("update");
        if (!drain) {
          throw new TRPCError({ code: "NOT_FOUND", message: "Log drain not found" });
        }

        await tx.delete(schema.logdrainState).where(eq(schema.logdrainState.logdrainId, drain.id));
        await tx
          .delete(schema.logdrains)
          .where(
            and(
              eq(schema.logdrains.id, input.id),
              eq(schema.logdrains.workspaceId, ctx.workspace.id),
            ),
          );
        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "logdrain.delete",
          description: `Deleted log drain ${input.id}`,
          resources: [],
          context: { location: ctx.audit.location, userAgent: ctx.audit.userAgent },
        });
      });
      return { id: input.id };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      console.error("Failed to delete log drain", error);
      throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: "Failed to delete log drain" });
    }
  });
