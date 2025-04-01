import { and, db, eq, isNull, schema, sql } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import {
  type RatelimitNamespacesResponse,
  queryRatelimitNamespacesPayload,
  ratelimitNamespacesResponse,
} from "./schemas";

export const queryRatelimitNamespaces = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(queryRatelimitNamespacesPayload)
  .output(ratelimitNamespacesResponse)
  .query(async ({ ctx, input }) => {
    try {
      const result = await fetchRatelimitNamespaces({
        workspaceId: ctx.workspace.id,
        limit: input.limit,
        cursor: input.cursor,
      });
      return result;
    } catch (error) {
      console.error(
        "Something went wrong when fetching ratelimit namespaces",
        JSON.stringify(error),
      );
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch ratelimit namespaces",
      });
    }
  });

export type RatelimitNamespaceOptions = {
  workspaceId: string;
  limit: number;
  cursor?: { id: string } | undefined;
};

export async function fetchRatelimitNamespaces({
  workspaceId,
  limit,
  cursor,
}: RatelimitNamespaceOptions): Promise<RatelimitNamespacesResponse> {
  // Get the total count of namespaces
  const totalResult = await db
    .select({ count: sql<number>`count(*)` })
    .from(schema.ratelimitNamespaces)
    .where(
      and(
        eq(schema.ratelimitNamespaces.workspaceId, workspaceId),
        isNull(schema.ratelimitNamespaces.deletedAtM),
      ),
    );

  const total = Number(totalResult[0]?.count || 0);

  // Query for ratelimit namespaces
  const query = db.query.ratelimitNamespaces.findMany({
    where: (table, { and, eq, isNull, gt }) => {
      const conditions = [eq(table.workspaceId, workspaceId), isNull(table.deletedAtM)];
      if (cursor) {
        conditions.push(gt(table.id, cursor.id));
      }
      return and(...conditions);
    },
    columns: {
      id: true,
      name: true,
    },
    orderBy: (table, { asc }) => [asc(table.id)],
    limit: limit + 1, // Fetch one extra to determine if there are more
  });

  const namespaces = await query;
  const hasMore = namespaces.length > limit;
  const namespaceItems = hasMore ? namespaces.slice(0, limit) : namespaces;
  const nextCursor =
    hasMore && namespaceItems.length > 0
      ? { id: namespaceItems[namespaceItems.length - 1].id }
      : undefined;

  return {
    namespaceList: namespaceItems,
    hasMore,
    nextCursor,
    total,
  };
}
