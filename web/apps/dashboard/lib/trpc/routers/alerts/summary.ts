import { and, count, db, eq, gte, schema } from "@/lib/db";
import { workspaceProcedure } from "../../trpc";

const sevenDaysMs = 7 * 24 * 60 * 60 * 1000;

export const getAlertsSummary = workspaceProcedure.query(async ({ ctx }) => {
  const [openRows, resolvedRows] = await Promise.all([
    db
      .select({ count: count() })
      .from(schema.alertEvents)
      .where(
        and(
          eq(schema.alertEvents.workspaceId, ctx.workspace.id),
          eq(schema.alertEvents.status, "open"),
        ),
      ),
    db
      .select({ count: count() })
      .from(schema.alertEvents)
      .where(
        and(
          eq(schema.alertEvents.workspaceId, ctx.workspace.id),
          eq(schema.alertEvents.status, "resolved"),
          gte(schema.alertEvents.resolvedAt, Date.now() - sevenDaysMs),
        ),
      ),
  ]);

  return {
    open: openRows[0]?.count ?? 0,
    resolvedLast7d: resolvedRows[0]?.count ?? 0,
  };
});
