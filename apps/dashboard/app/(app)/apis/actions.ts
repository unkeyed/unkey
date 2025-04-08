"use server";
import { and, db, eq, isNull, schema, sql } from "@/lib/db";
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
          id: true, // Include the keyspace ID
          sizeApprox: true,
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

  // Transform the data to include key information
  const apiList = await Promise.all(
    apiItems.map(async (api) => {
      const keyspaceId = api.keyAuth?.id || null;

      return {
        id: api.id,
        name: api.name,
        keyspaceId,
        keys: [
          {
            count: api.keyAuth?.sizeApprox || 0,
          },
        ],
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

export async function apiItemsWithApproxKeyCounts(apiItems: Array<any>) {
  return apiItems.map((api) => {
    return {
      id: api.id,
      name: api.name,
      keyspaceId: api.keyAuthId,
      keys: [{ count: api.keyAuth?.sizeApprox || 0 }],
    };
  });
}
