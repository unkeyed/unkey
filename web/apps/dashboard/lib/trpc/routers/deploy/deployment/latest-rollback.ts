import type { User } from "@/lib/auth/types";
import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { getCachedUser } from "@/lib/trpc/routers/audit/fetch";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { AUDIT_LOG_BUCKET_UNKEY_MUTATIONS, unkeyAuditLogEvents } from "@unkey/schema/src/auditlog";
import { z } from "zod";

export const getLatestRollback = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ appId: z.string(), deploymentId: z.string() }))
  .query(async ({ ctx, input }) => {
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
    // Target order is set by svc/ctrl rollback_handler.go: rolled-back-to, app, replaced.
    const [to, from] = (row?.targets ?? []).filter(([type]) => type === "deployment");
    if (!row || to?.[1] !== input.deploymentId) {
      return null;
    }
    const user = row.actorType === "user" ? await getCachedUser(row.actorId) : null;
    return {
      at: row.time,
      actor: actorOf(user, row.actorName),
      fromDeploymentId: from?.[1] ?? null,
    };
  });

function actorOf(
  user: User | null,
  actorName: string,
): { name: string; avatarUrl: string | null } | null {
  if (user) {
    const name = `${user.firstName ?? ""} ${user.lastName ?? ""}`.trim() || user.email;
    return { name, avatarUrl: user.avatarUrl ?? null };
  }
  if (actorName) {
    return { name: actorName, avatarUrl: null };
  }
  return null;
}
