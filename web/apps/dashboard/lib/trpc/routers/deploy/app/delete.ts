import { insertAuditLogs } from "@/lib/audit";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { getCtrlClients } from "../../ctrl";

export const deleteApp = workspaceProcedure
  .input(z.object({ appId: z.string() }))
  .use(withRatelimit(ratelimit.delete))
  .mutation(async ({ ctx, input }) => {
    const app = await db.query.apps.findFirst({
      where: (table, { and, eq }) =>
        and(eq(table.id, input.appId), eq(table.workspaceId, ctx.workspace.id)),
    });

    if (!app) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "App not found",
      });
    }

    if (app.deleteProtection) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Cannot delete app with delete protection enabled",
      });
    }

    const ctrl = getCtrlClients();

    try {
      await ctrl.app.deleteApp({
        appId: input.appId,
      });
    } catch (err) {
      if (err instanceof TRPCError) {
        throw err;
      }
      console.error("Failed to delete app:", err);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to delete app",
      });
    }

    await insertAuditLogs(db, {
      workspaceId: ctx.workspace.id,
      actor: { type: "user", id: ctx.user.id },
      event: "app.delete",
      description: `Deleted ${app.id}`,
      resources: [
        {
          type: "app",
          id: app.id,
          name: app.name,
        },
      ],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });

    return { success: true, appId: input.appId };
  });
