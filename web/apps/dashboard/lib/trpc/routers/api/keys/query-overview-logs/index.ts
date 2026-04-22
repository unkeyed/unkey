import { keysQueryOverviewLogsPayload } from "@/components/api-requests-table/schema/keys-overview.schema";
import { clickhouse } from "@/lib/clickhouse";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { keysOverviewLogs as keysLogs } from "@unkey/clickhouse/src/keys/keys";
import { z } from "zod";
import { createKeyDetailsMap, getApi, queryApiKeys } from "../api-query";
import { transformKeysFilters } from "./utils";

const KeysOverviewLogsResponse = z.object({
  keysOverviewLogs: z.array(keysLogs),
  hasMore: z.boolean(),
  nextCursor: z.int().optional(),
  total: z.int(),
});

type KeysOverviewLogsResponse = z.infer<typeof KeysOverviewLogsResponse>;

/**
 * This procedure queries keys overview logs by:
 * 1. First querying ClickHouse with relevant filters
 * 2. Then filtering the results with SQL
 * 3. Finally merging with key details
 */
export const queryKeysOverviewLogs = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .meta({ skipBatch: true })
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

    // Name/identity filters live in Postgres, not ClickHouse (the raw table
    // has no `name` column). Resolve them to a concrete keyId set up front
    // so the CH query's pagination and total_count reflect the filter.
    // Without this, CH would paginate/count ignoring name/identity, and the
    // Postgres post-filter below would silently drop rows after pagination
    // had already been committed to — producing half-empty pages and wrong
    // totals.
    const hasMetadataFilter = (input.names?.length ?? 0) > 0 || (input.identities?.length ?? 0) > 0;

    let keyIdsForClickhouse = input.keyIds ? transformedInputs.keyIds : null;
    let preResolvedKeys: Awaited<ReturnType<typeof queryApiKeys>>["keys"] | null = null;

    if (hasMetadataFilter) {
      const result = await queryApiKeys({
        apiId: input.apiId,
        workspaceId: ctx.workspace.id,
        keyIds: input.keyIds ?? null,
        names: input.names ?? null,
        identities: input.identities ?? null,
      });
      preResolvedKeys = result.keys;

      if (preResolvedKeys.length === 0) {
        return {
          keysOverviewLogs: [],
          hasMore: false,
          total: 0,
        };
      }

      keyIdsForClickhouse = preResolvedKeys.map((k) => ({
        operator: "is" as const,
        value: k.id,
      }));
    }

    const { logsQuery, countQuery } = await clickhouse.api.keys.logs({
      ...transformedInputs,
      workspaceId: ctx.workspace.id,
      keyspaceId: keyspaceId,
      useTimeFrameFilter: input.useTimeFrameFilter ?? false,
      keyIds: keyIdsForClickhouse,
      tags: transformedInputs.tags,
      // Applied via keyIdsForClickhouse above; CH has no name column.
      names: null,
      identities: null,
    });

    const [clickhouseResult, countResult] = await Promise.all([logsQuery, countQuery]);

    if (!clickhouseResult || clickhouseResult.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching data from ClickHouse.",
      });
    }

    if (countResult.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching count from ClickHouse.",
      });
    }

    const logs = clickhouseResult.val || [];
    const totalCount = countResult.val[0]?.total_count ?? 0;

    if (logs.length === 0) {
      return {
        keysOverviewLogs: [],
        hasMore: false,
        total: 0,
      };
    }

    // Hydrate key details. If we pre-resolved keys for the metadata filter,
    // reuse them — no second Postgres round trip. Otherwise look up the page
    // of keyIds CH just returned.
    const keys =
      preResolvedKeys ??
      (
        await queryApiKeys({
          apiId: input.apiId,
          workspaceId: ctx.workspace.id,
          keyIds: logs.map((log) => ({ operator: "is" as const, value: log.key_id })),
          names: null,
          identities: null,
        })
      ).keys;

    const keyDetailsMap = createKeyDetailsMap(keys);

    // With a metadata filter, CH is already scoped to the pre-resolved keyIds;
    // filter defensively in case CH returns a stale key_id. Without a
    // metadata filter, include every log — a key that was soft-deleted after
    // generating usage still has legitimate activity, and dropping it here
    // would make the table totals disagree with the chart totals.
    const keysOverviewLogs = (
      hasMetadataFilter ? logs.filter((log) => keyDetailsMap.has(log.key_id)) : logs
    ).map((log) => ({
      ...log,
      key_details: keyDetailsMap.get(log.key_id) ?? null,
    }));

    const hasMore = logs.length === input.limit && keysOverviewLogs.length > 0;
    const response: KeysOverviewLogsResponse = {
      keysOverviewLogs,
      hasMore,
      nextCursor: hasMore ? logs[logs.length - 1].time : undefined,
      total: totalCount,
    };

    return response;
  });
