import { type SQL, and, db, desc, eq, gt, lt, or, schema, sql } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { workspaceProcedure } from "../../trpc";
import { listAlertsInput } from "./schemas";
import { alertSelection } from "./selection";

const statusOrder = sql<number>`CASE ${schema.alertEvents.status} WHEN 'open' THEN 0 ELSE 1 END`;

export const listAlerts = workspaceProcedure
  .input(listAlertsInput)
  .query(async ({ ctx, input }) => {
    const filters: SQL[] = [eq(schema.alertEvents.workspaceId, ctx.workspace.id)];
    if (input.status !== "all") {
      filters.push(eq(schema.alertEvents.status, input.status));
    }
    if (input.metric) {
      filters.push(eq(schema.alertEvents.metric, input.metric));
    }
    if (input.appId) {
      filters.push(eq(schema.alertEvents.appId, input.appId));
    }
    if (input.environmentId) {
      filters.push(eq(schema.alertEvents.environmentId, input.environmentId));
    }
    if (input.startMs !== undefined) {
      filters.push(gt(schema.alertEvents.windowEnd, input.startMs));
    }
    if (input.endMs !== undefined) {
      filters.push(lt(schema.alertEvents.windowStart, input.endMs));
    }

    if (input.cursor) {
      const [cursor] = await db
        .select({
          pk: schema.alertEvents.pk,
          status: schema.alertEvents.status,
          firedAt: schema.alertEvents.firedAt,
        })
        .from(schema.alertEvents)
        .where(and(...filters, eq(schema.alertEvents.id, input.cursor)));
      if (!cursor) {
        throw new TRPCError({ code: "BAD_REQUEST", message: "Invalid alert cursor" });
      }

      filters.push(
        or(
          gt(statusOrder, cursor.status === "open" ? 0 : 1),
          and(
            eq(schema.alertEvents.status, cursor.status),
            or(
              lt(schema.alertEvents.firedAt, cursor.firedAt),
              and(
                eq(schema.alertEvents.firedAt, cursor.firedAt),
                lt(schema.alertEvents.pk, cursor.pk),
              ),
            ),
          ),
        ) ?? sql`false`,
      );
    }

    const rows = await db
      .select({ pk: schema.alertEvents.pk, ...alertSelection })
      .from(schema.alertEvents)
      .innerJoin(schema.apps, eq(schema.apps.id, schema.alertEvents.appId))
      .innerJoin(schema.environments, eq(schema.environments.id, schema.alertEvents.environmentId))
      .innerJoin(schema.projects, eq(schema.projects.id, schema.alertEvents.projectId))
      .where(and(...filters))
      .orderBy(statusOrder, desc(schema.alertEvents.firedAt), desc(schema.alertEvents.pk))
      .limit(input.limit + 1);

    const hasMore = rows.length > input.limit;
    const page = hasMore ? rows.slice(0, input.limit) : rows;
    const alerts = page.map(({ pk: _, ...alert }) => alert);

    return {
      alerts,
      nextCursor: hasMore ? alerts.at(-1)?.id : undefined,
    };
  });
