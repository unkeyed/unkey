import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { workspaceProcedure } from "../../trpc";
import { resolveAlertInput } from "./schemas";

export const resolveAlert = workspaceProcedure
  .input(resolveAlertInput)
  .mutation(async ({ ctx, input }) => {
    return db.transaction(async (tx) => {
      const [alert] = await tx
        .select({
          id: schema.alertEvents.id,
          status: schema.alertEvents.status,
          appId: schema.alertEvents.appId,
          environmentId: schema.alertEvents.environmentId,
          metric: schema.alertEvents.metric,
        })
        .from(schema.alertEvents)
        .where(
          and(
            eq(schema.alertEvents.id, input.alertId),
            eq(schema.alertEvents.workspaceId, ctx.workspace.id),
          ),
        )
        .for("update");

      if (!alert) {
        throw new TRPCError({ code: "NOT_FOUND", message: "Alert not found" });
      }
      if (alert.status === "resolved") {
        throw new TRPCError({
          code: "PRECONDITION_FAILED",
          message: "Alert is already resolved",
        });
      }

      const resolvedAt = Date.now();
      await tx
        .update(schema.alertEvents)
        .set({
          status: "resolved",
          resolvedAt,
          resolvedBy: ctx.user.id,
          resolutionMessage: input.message,
          updatedAt: resolvedAt,
        })
        .where(
          and(
            eq(schema.alertEvents.id, alert.id),
            eq(schema.alertEvents.workspaceId, ctx.workspace.id),
          ),
        );

      await insertAuditLogs(tx, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "alert.resolve",
        description: `Resolved ${alert.metric} alert ${alert.id}`,
        resources: [{ type: "alert", id: alert.id }],
        context: ctx.audit,
      });

      return {
        id: alert.id,
        status: "resolved" as const,
        resolvedAt,
        resolvedBy: ctx.user.id,
        resolutionMessage: input.message,
      };
    });
  });
