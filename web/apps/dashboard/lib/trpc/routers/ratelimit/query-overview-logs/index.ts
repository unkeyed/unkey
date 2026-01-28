import { ratelimitQueryOverviewLogsPayload } from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/_overview/components/table/query-logs.schema";
import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { type RatelimitOverviewLog, ratelimitOverviewLogs } from "@unkey/clickhouse/src/ratelimits";
import { z } from "zod";
import { transformFilters } from "./utils";

const RatelimitOverviewLogsResponse = z.object({
  ratelimitOverviewLogs: z.array(ratelimitOverviewLogs),
  hasMore: z.boolean(),
  total: z.number(),
  nextCursor: z.int().optional(),
});

type RatelimitOverviewLogsResponse = z.infer<typeof RatelimitOverviewLogsResponse>;

export const queryRatelimitOverviewLogs = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(ratelimitQueryOverviewLogsPayload)
  .output(RatelimitOverviewLogsResponse)
  .query(async ({ ctx, input }) => {
    const ratelimitNamespace = await db.query.ratelimitNamespaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(
            eq(table.workspaceId, ctx.workspace.id),
            eq(table.id, input.namespaceId),
            isNull(table.deletedAtM),
          ),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve ratelimit namespace. If this issue persists, please contact support@unkey.dev",
        });
      });

    if (!ratelimitNamespace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Namespace not found",
      });
    }

    const transformedInputs = transformFilters(input);
    const { countQuery, logsQuery } = await clickhouse.ratelimits.overview.logs({
      ...transformedInputs,
      cursorTime: input.cursor ?? null,
      workspaceId: ctx.workspace.id,
      namespaceId: ratelimitNamespace.id,
    });

    const [countResult, logsResult] = await Promise.all([countQuery, logsQuery]);

    if (countResult.err || logsResult.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching data from clickhouse.",
      });
    }

    const logsWithOverrides = await checkIfIdentifierHasOverride(
      logsResult.val,
      ratelimitNamespace.id,
      ctx.workspace.id,
    );

    const response: RatelimitOverviewLogsResponse = {
      ratelimitOverviewLogs: logsWithOverrides,
      total: countResult.val[0].total_count,
      hasMore: logsWithOverrides.length === input.limit,
      nextCursor:
        logsWithOverrides.length === input.limit
          ? logsWithOverrides[logsWithOverrides.length - 1].time
          : undefined,
    };

    return response;
  });

async function checkIfIdentifierHasOverride(
  logs: RatelimitOverviewLog[],
  namespaceId: string,
  workspaceId: string,
) {
  const identifiers = [...new Set(logs.map((log) => log.identifier))];

  if (identifiers.length === 0) {
    return logs.map((log) => ({
      ...log,
      override: null,
    }));
  }

  const overrides = await db.query.ratelimitOverrides
    .findMany({
      where: (table, { and, isNull, inArray, eq }) =>
        and(
          eq(table.namespaceId, namespaceId),
          eq(table.workspaceId, workspaceId),
          inArray(table.identifier, identifiers),
          isNull(table.deletedAtM),
        ),
      columns: {
        identifier: true,
        limit: true,
        duration: true,
        id: true,
        async: true,
      },
    })
    .catch((_err) => {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to retrieve rate limit overrides. Please try again later.",
      });
    });

  const overrideMap = new Map(
    overrides.map((override) => [
      override.identifier,
      {
        limit: override.limit,
        duration: override.duration,
        overrideId: override.id,
        async: override.async,
      },
    ]),
  );

  const logsWithOverrides = logs.map((log) => ({
    ...log,
    override: overrideMap.get(log.identifier) || null,
  }));

  return logsWithOverrides;
}
