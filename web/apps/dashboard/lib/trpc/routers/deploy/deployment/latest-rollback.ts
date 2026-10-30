import { getCachedUser } from "@/lib/auth/cached-user";
import { clickhouse } from "@/lib/clickhouse";
import type { RollbackEvent } from "@/lib/collections/deploy/rollback";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { AUDIT_LOG_BUCKET_UNKEY_MUTATIONS, unkeyAuditLogEvents } from "@unkey/schema/src/auditlog";
import { z } from "zod";
import { toRollbackEvent } from "./rollback-event";

export const getLatestRollback = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ appId: z.string(), deploymentId: z.string() }))
  .query(async ({ ctx, input }): Promise<RollbackEvent | null> => {
    const app = await db.query.apps.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(
          eq(table.id, input.appId),
          eq(table.workspaceId, ctx.workspace.id),
          isNull(table.deletedAt),
        ),
      columns: { currentDeploymentId: true, isRolledBack: true },
    });
    if (!app) {
      throw new TRPCError({ code: "NOT_FOUND", message: "App not found" });
    }
    if (!app.isRolledBack || app.currentDeploymentId !== input.deploymentId) {
      return null;
    }
    const result = await clickhouse.auditLogs.latestByTarget({
      workspaceId: ctx.workspace.id,
      bucket: AUDIT_LOG_BUCKET_UNKEY_MUTATIONS,
      event: unkeyAuditLogEvents.enum["deployment.rollback"],
      targetId: input.appId,
    });
    if (result.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to load the rollback event.",
        cause: result.err,
      });
    }
    const row = result.val[0];
    if (!row) {
      return null;
    }

    const user = row.actorType === "user" ? await getCachedUser(row.actorId) : null;
    return toRollbackEvent(row, user, input.deploymentId);
  });
