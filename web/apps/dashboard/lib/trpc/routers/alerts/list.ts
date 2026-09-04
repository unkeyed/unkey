import { type SQL, and, db, desc, eq, gt, lt, or, schema, sql } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";
import { listAlertsInput } from "./schemas";
import { alertSelection } from "./selection";

const alertCursor = z.object({
  firedAt: z.number().int().nonnegative(),
  pk: z.number().int().positive(),
});

function encodeAlertCursor(cursor: z.infer<typeof alertCursor>): string {
  return Buffer.from(JSON.stringify(cursor)).toString("base64url");
}

function decodeAlertCursor(cursor: string): z.infer<typeof alertCursor> {
  try {
    const decoded: unknown = JSON.parse(Buffer.from(cursor, "base64url").toString("utf8"));
    return alertCursor.parse(decoded);
  } catch {
    throw new TRPCError({ code: "BAD_REQUEST", message: "Invalid alert cursor" });
  }
}

export const listAlerts = workspaceProcedure
  .input(listAlertsInput)
  .query(async ({ ctx, input }) => {
    const filters: SQL[] = [eq(schema.alertEvents.workspaceId, ctx.workspace.id)];
    if (!input.includeResolved) {
      filters.push(eq(schema.alertEvents.status, "open"));
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
      const cursor = decodeAlertCursor(input.cursor);

      filters.push(
        or(
          lt(schema.alertEvents.firedAt, cursor.firedAt),
          and(eq(schema.alertEvents.firedAt, cursor.firedAt), lt(schema.alertEvents.pk, cursor.pk)),
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
      .orderBy(desc(schema.alertEvents.firedAt), desc(schema.alertEvents.pk))
      .limit(input.limit + 1);

    const hasMore = rows.length > input.limit;
    const page = hasMore ? rows.slice(0, input.limit) : rows;
    const alerts = page.map(({ pk: _, ...alert }) => alert);
    const boundary = hasMore ? page.at(-1) : undefined;

    return {
      alerts,
      nextCursor: boundary
        ? encodeAlertCursor({ firedAt: boundary.firedAt, pk: boundary.pk })
        : undefined,
    };
  });
