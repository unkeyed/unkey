"use server";
import { and, count, db, eq, inArray, isNull, schema, sql } from "@/lib/db";
import type { ApisOverviewResponse } from "@/lib/trpc/routers/api/overview/query-overview/schemas";

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

  // Updated query to include keyAuth and fetch actual keys
  const query = db.query.apis.findMany({
    where: (table, { and, eq, isNull, gt }) => {
      const conditions = [eq(table.workspaceId, workspaceId), isNull(table.deletedAtM)];
      if (cursor) {
        conditions.push(gt(table.id, cursor.id));
      }
      return and(...conditions);
    },
    with: {
      keyAuth: {
        columns: {
          id: true,
        },
      },
    },
    orderBy: (table, { asc }) => [asc(table.id)],
    limit: limit + 1, // Fetch one extra to determine if there are more
  });

  const apis = await query;
  const hasMore = apis.length > limit;
  const apiItems = hasMore ? apis.slice(0, limit) : apis;
  const nextCursor =
    hasMore && apiItems.length > 0 ? { id: apiItems[apiItems.length - 1].id } : undefined;

  const apiList = await attachKeyCounts(
    workspaceId,
    apiItems.map((api) => ({ id: api.id, name: api.name, keyAuthId: api.keyAuth?.id ?? null })),
  );

  return {
    apiList,
    hasMore,
    nextCursor,
    total,
  };
}

type ApiItem = {
  id: string;
  name: string;
  keyAuthId: string | null;
};

type ApiWithKeyCount = {
  id: string;
  name: string;
  keyspaceId: string | null;
  keyCount: number;
};

export async function attachKeyCounts(
  workspaceId: string,
  apiItems: Array<ApiItem>,
): Promise<Array<ApiWithKeyCount>> {
  const keyAuthIds = apiItems
    .map((api) => api.keyAuthId)
    .filter((id): id is string => Boolean(id));

  const keyCountsByKeyAuthId = new Map<string, number>();
  if (keyAuthIds.length > 0) {
    const rows = await db
      .select({
        keyAuthId: schema.keys.keyAuthId,
        count: count(schema.keys.id),
      })
      .from(schema.keys)
      .where(
        and(
          eq(schema.keys.workspaceId, workspaceId),
          inArray(schema.keys.keyAuthId, keyAuthIds),
          isNull(schema.keys.deletedAtM),
        ),
      )
      .groupBy(schema.keys.keyAuthId);

    for (const row of rows) {
      keyCountsByKeyAuthId.set(row.keyAuthId, Number(row.count));
    }
  }

  return apiItems.map((api) => ({
    id: api.id,
    name: api.name,
    keyspaceId: api.keyAuthId,
    keyCount: api.keyAuthId ? (keyCountsByKeyAuthId.get(api.keyAuthId) ?? 0) : 0,
  }));
}
