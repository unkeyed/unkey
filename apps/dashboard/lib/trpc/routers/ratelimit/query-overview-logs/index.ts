import { ratelimitQueryOverviewLogsPayload } from "@/app/(app)/ratelimits/[namespaceId]/_overview/components/table/query-logs.schema";
import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { type RatelimitOverviewLog, ratelimitOverviewLogs } from "@unkey/clickhouse/src/ratelimits";
import { z } from "zod";
import { transformFilters } from "./utils";

const RatelimitOverviewLogsResponse = z.object({
  ratelimitOverviewLogs: z.array(ratelimitOverviewLogs),
  hasMore: z.boolean(),
  nextCursor: z
    .object({
      time: z.number().int(),
      requestId: z.string(),
    })
    .optional(),
});

type RatelimitOverviewLogsResponse = z.infer<typeof RatelimitOverviewLogsResponse>;

export const queryRatelimitOverviewLogs = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(ratelimitQueryOverviewLogsPayload)
  .output(RatelimitOverviewLogsResponse)
  .query(async ({ ctx, input }) => {
    const ratelimitNamespaces = await db.query.ratelimitNamespaces
      .findMany({
        where: (table, { and, eq, isNull }) =>
          and(
            eq(table.workspaceId, ctx.workspace.id),
            and(eq(table.id, input.namespaceId), isNull(table.deletedAtM)),
          ),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve ratelimit timeseries analytics due to a workspace error. If this issue persists, please contact support@unkey.dev with the time this occurred.",
        });
      });

    if (!ratelimitNamespaces) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Ratelimit namespaces not found, please contact support using support@unkey.dev.",
      });
    }

    if (ratelimitNamespaces.length === 0) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Namespace not found",
      });
    }

    const transformedInputs = transformFilters(input);
    const result = await clickhouse.ratelimits.overview.logs({
      ...transformedInputs,
      cursorRequestId: input.cursor?.requestId ?? null,
      cursorTime: input.cursor?.time ?? null,
      workspaceId: ctx.workspace.id,
      namespaceId: ratelimitNamespaces[0].id,
    });

    if (result.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching data from clickhouse.",
      });
    }

    const logsWithOverrides = await checkIfIdentifierHasOverride(result.val);

    const response: RatelimitOverviewLogsResponse = {
      ratelimitOverviewLogs: logsWithOverrides,
      hasMore: logsWithOverrides.length === input.limit,
      nextCursor:
        logsWithOverrides.length === input.limit
          ? {
              time: logsWithOverrides[logsWithOverrides.length - 1].time,
              requestId: logsWithOverrides[logsWithOverrides.length - 1].request_id,
            }
          : undefined,
    };

    return response;
  });

async function checkIfIdentifierHasOverride(logs: RatelimitOverviewLog[]) {
  const identifiers = [...new Set(logs.map((log) => log.identifier))];

  // if there are no identifiers to check
  if (identifiers.length === 0) {
    return logs.map((log) => ({
      ...log,
      override: null,
    }));
  }

  const overrides = await db.query.ratelimitOverrides
    .findMany({
      where: (table, { and, isNull, inArray }) =>
        and(inArray(table.identifier, identifiers), isNull(table.deletedAtM)),
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
