import { keysQueryOverviewLogsPayload } from "@/app/(app)/apis/[apiId]/_overview/components/table/query-logs.schema";
import { clickhouse } from "@/lib/clickhouse";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { keysOverviewLogs as keysLogs } from "@unkey/clickhouse/src/keys/keys";
import { z } from "zod";
import { createKeyDetailsMap, getApi, queryApiKeys } from "../api-query";
import { transformKeysFilters } from "./utils";

const KeysOverviewLogsResponse = z.object({
  keysOverviewLogs: z.array(keysLogs),
  hasMore: z.boolean(),
  nextCursor: z
    .object({
      time: z.number().int(),
      requestId: z.string(),
    })
    .optional(),
});

type KeysOverviewLogsResponse = z.infer<typeof KeysOverviewLogsResponse>;

/**
 * This procedure queries keys overview logs by:
 * 1. First querying ClickHouse with relevant filters
 * 2. Then filtering the results with SQL
 * 3. Finally merging with key details
 */
export const queryKeysOverviewLogs = rateLimitedProcedure(ratelimit.read)
  .input(keysQueryOverviewLogsPayload)
  .output(KeysOverviewLogsResponse)
  .query(async ({ ctx, input }) => {
    const api = await getApi(input.apiId, ctx.workspace.id);
    if (!api || !api.keyAuth?.id) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "API not found or does not have key authentication enabled",
      });
    }
    const keyspaceId = api.keyAuth.id;

    const transformedInputs = transformKeysFilters(input);

    const clickhouseResult = await clickhouse.api.keys.logs({
      ...transformedInputs,
      cursorRequestId: input.cursor?.requestId ?? null,
      cursorTime: input.cursor?.time ?? null,
      workspaceId: ctx.workspace.id,
      keyspaceId: keyspaceId,
      // Only include keyIds filters if explicitly provided in the input
      keyIds: input.keyIds ? transformedInputs.keyIds : null,
      // Nullify these as we'll filter in the database
      names: null,
      identities: null,
    });

    if (!clickhouseResult || clickhouseResult.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching data from ClickHouse.",
      });
    }

    const logs = clickhouseResult.val || [];
    if (logs.length === 0) {
      return {
        keysOverviewLogs: [],
        hasMore: false,
      };
    }

    const keyIdsFromLogs = logs.map((log) => log.key_id);

    // This ensures we only get keys that exist in both ClickHouse and the database
    const { keys } = await queryApiKeys({
      apiId: input.apiId,
      workspaceId: ctx.workspace.id,
      // Pass the key IDs from ClickHouse logs as "is" filters
      keyIds: keyIdsFromLogs.map((id) => ({ operator: "is", value: id })),
      // Still apply any name or identity filters from the original input
      names: input.names || null,
      identities: input.identities || null,
    });

    const keyDetailsMap = createKeyDetailsMap(keys);
    const filteredKeyIds = Array.from(keyDetailsMap.keys());

    // Only include logs for keys that exist in the database and passed all filters
    const keysOverviewLogs = logs
      .filter((log) => filteredKeyIds.includes(log.key_id))
      .map((log) => ({
        ...log,
        key_details: keyDetailsMap.get(log.key_id) || null,
      }));

    const response: KeysOverviewLogsResponse = {
      keysOverviewLogs,
      hasMore: logs.length === input.limit && keysOverviewLogs.length > 0,
      nextCursor:
        logs.length === input.limit
          ? {
              time: logs[logs.length - 1].time,
              requestId: logs[logs.length - 1].request_id,
            }
          : undefined,
    };

    return response;
  });
