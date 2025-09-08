import {
  type CursorType,
  type NamespaceListInputSchema,
  type NamespaceListOutputSchema,
  namespaceListInputSchema,
  namespaceListOutputSchema,
} from "@/app/(app)/[workspace]/ratelimits/_components/list/namespace-list.schema";
import { and, db, desc, eq, isNull, like, lt, schema, sql } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";

const LIMIT = 9;
export const queryRatelimitNamespaces = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(namespaceListInputSchema)
  .output(namespaceListOutputSchema)
  .query(async ({ ctx, input }) => {
    try {
      const result = await fetchRatelimitNamespaces({
        workspaceId: ctx.workspace.id,
        limit: LIMIT,
        cursor: input.cursor,
        nameQuery: input.query,
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
  cursor?: CursorType;
  nameQuery?: NamespaceListInputSchema["query"];
};

export async function fetchRatelimitNamespaces({
  workspaceId,
  limit,
  cursor,
  nameQuery,
}: RatelimitNamespaceOptions): Promise<NamespaceListOutputSchema> {
  // Build base conditions
  const baseConditions = [
    eq(schema.ratelimitNamespaces.workspaceId, workspaceId),
    isNull(schema.ratelimitNamespaces.deletedAtM),
  ];

  // Add name query conditions
  if (nameQuery?.length) {
    const filter = nameQuery[0];
    if (filter.operator === "contains") {
      baseConditions.push(like(schema.ratelimitNamespaces.name, `%${filter.value}%`));
    }
  }

  const whereClause = and(...baseConditions);

  // Get total count
  const totalResult = await db
    .select({ count: sql<number>`count(*)` })
    .from(schema.ratelimitNamespaces)
    .where(whereClause);

  const total = Number(totalResult[0]?.count || 0);

  // Build query conditions (includes cursor)
  const queryConditions = [...baseConditions];

  if (cursor) {
    queryConditions.push(lt(schema.ratelimitNamespaces.id, cursor.id));
  }

  const namespaces = await db
    .select({
      id: schema.ratelimitNamespaces.id,
      name: schema.ratelimitNamespaces.name,
    })
    .from(schema.ratelimitNamespaces)
    .where(and(...queryConditions))
    .orderBy(desc(schema.ratelimitNamespaces.id))
    .limit(limit + 1);

  const hasMore = namespaces.length > limit;
  const namespaceItems = hasMore ? namespaces.slice(0, limit) : namespaces;

  const nextCursor: CursorType | undefined =
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
