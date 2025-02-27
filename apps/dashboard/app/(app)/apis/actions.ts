"use server";

import { and, db, eq, isNull, schema, sql } from "@/lib/db";
import type { ApisOverviewResponse } from "@/lib/trpc/routers/api/overview/schemas";

export type ApiOverviewOptions = {
  workspaceId: string;
  limit: number;
  cursor?: { id: string } | undefined;
};

export async function fetchApiOverview({
  workspaceId,
  limit,
  cursor,
}: ApiOverviewOptions): Promise<ApisOverviewResponse> {
  const totalResult = await db
    .select({ count: sql<number>`count(*)` })
    .from(schema.apis)
    .where(and(eq(schema.apis.workspaceId, workspaceId), isNull(schema.apis.deletedAtM)));
  const total = Number(totalResult[0]?.count || 0);

  const query = db.query.apis.findMany({
    where: (table, { and, eq, isNull, gt }) => {
      const conditions = [eq(table.workspaceId, workspaceId), isNull(table.deletedAtM)];
      if (cursor) {
        conditions.push(gt(table.id, cursor.id));
      }
      return and(...conditions);
    },
    orderBy: (table, { asc }) => [asc(table.id)],
    limit: Number(limit) + 1, // Fetch one extra to determine if there are more
  });

  const apis = await query;
  const hasMore = apis.length > limit;
  const apiItems = hasMore ? apis.slice(0, limit) : apis;

  const nextCursor =
    hasMore && apiItems.length > 0 ? { id: apiItems[apiItems.length - 1].id } : undefined;

  const apiList = await Promise.all(
    apiItems.map(async (api) => {
      const keyCountResult = await db
        .select({ count: sql<number>`count(*)` })
        .from(schema.keys)
        .where(and(eq(schema.keys.keyAuthId, api.keyAuthId!), isNull(schema.keys.deletedAtM)));
      const keyCount = Number(keyCountResult[0]?.count || 0);

      return {
        id: api.id,
        name: api.name,
        keyspaceId: api.keyAuthId,
        keys: [{ count: keyCount }],
      };
    }),
  );

  return {
    apiList,
    hasMore,
    nextCursor,
    total,
  };
}
