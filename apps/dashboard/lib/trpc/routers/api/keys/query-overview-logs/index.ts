import { keysQueryOverviewLogsPayload } from "@/app/(app)/apis/[apiId]/_overview/components/table/query-logs.schema";
import { clickhouse } from "@/lib/clickhouse";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { keysOverviewLogs as keysLogs } from "@unkey/clickhouse/src/keys/keys";
import { z } from "zod";
import { createKeyDetailsMap, queryApiKeys } from "../api-query";
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

export const queryKeysOverviewLogs = rateLimitedProcedure(ratelimit.read)
  .input(keysQueryOverviewLogsPayload)
  .output(KeysOverviewLogsResponse)
  .query(async ({ ctx, input }) => {
    const transformedInputs = transformKeysFilters(input);

    const { keyspaceId, keys, keyIds } = await queryApiKeys({
      apiId: input.apiId,
      workspaceId: ctx.workspace.id,
      keyIds: transformedInputs.keyIds,
      names: transformedInputs.names,
      identities: transformedInputs.identities,
    });

    const keyDetailsMap = createKeyDetailsMap(keys);

    const result = await clickhouse.api.keys.logs({
      ...transformedInputs,
      cursorRequestId: input.cursor?.requestId ?? null,
      cursorTime: input.cursor?.time ?? null,
      workspaceId: ctx.workspace.id,
      keyspaceId: keyspaceId,
      keyIds: (keyIds ?? []).map((x) => ({
        value: String(x.value),
        operator: x.operator,
      })),
    });

    if (!result || result.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching data from clickhouse.",
      });
    }

    const logs = result.val || [];
    if (logs.length === 0) {
      return {
        keysOverviewLogs: [],
        hasMore: false,
      };
    }

    const keysOverviewLogs = logs.map((log) => ({
      ...log,
      key_details: keyDetailsMap.get(log.key_id) || null,
    }));

    const response: KeysOverviewLogsResponse = {
      keysOverviewLogs,
      hasMore: logs.length === input.limit,
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
