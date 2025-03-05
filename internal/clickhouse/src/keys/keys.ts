import { z } from "zod";
import type { Querier } from "../client";

export const keysOverviewLogsParams = z.object({
  workspaceId: z.string(),
  keyspaceId: z.string(),
  limit: z.number().int(),
  startTime: z.number().int(),
  endTime: z.number().int(),
  outcomes: z
    .array(
      z.object({
        value: z.enum([
          "VALID",
          "INSUFFICIENT_PERMISSIONS",
          "RATE_LIMITED",
          "FORBIDDEN",
          "DISABLED",
          "EXPIRED",
          "USAGE_EXCEEDED",
          "",
        ]),
        operator: z.literal("is"),
      })
    )
    .nullable(),
  keyIds: z
    .array(
      z.object({
        operator: z.enum(["is", "contains"]),
        value: z.string(),
      })
    )
    .nullable(),
  cursorTime: z.number().int().nullable(),
  cursorRequestId: z.string().nullable(),
  sorts: z
    .array(
      z.object({
        column: z.enum(["time", "valid_count", "error_count"]),
        direction: z.enum(["asc", "desc"]),
      })
    )
    .nullable(),
});

export const keyDetailsResponseSchema = z.object({
  id: z.string(),
  key_auth_id: z.string(),
  name: z.string().nullable(),
  owner_id: z.string().nullable(),
  identity_id: z.string().nullable(),
  meta: z.string().nullable(),
  enabled: z.boolean(),
  remaining_requests: z.number().nullable(),
  ratelimit_async: z.boolean().nullable(),
  ratelimit_limit: z.number().nullable(),
  ratelimit_duration: z.number().nullable(),
  environment: z.string().nullable(),
  refill_day: z.number().nullable(),
  refill_amount: z.number().nullable(),
  last_refill_at: z.date().nullable(),
  expires: z.date().nullable(),
  workspace_id: z.string(),
});

export type KeyDetailsResponse = z.infer<typeof keyDetailsResponseSchema>;

export const rawKeysOverviewLogs = z.object({
  time: z.number().int(),
  key_id: z.string(),
  request_id: z.string(),
  valid_count: z.number().int(),
  error_count: z.number().int(),
  outcome_counts: z.record(z.string(), z.number().int()),
});

export const keysOverviewLogs = rawKeysOverviewLogs.extend({
  key_details: keyDetailsResponseSchema.nullable(),
});

export type RawKeysOverviewLog = z.infer<typeof rawKeysOverviewLogs>;
export type KeysOverviewLog = z.infer<typeof keysOverviewLogs>;
export type KeysOverviewLogsParams = z.infer<typeof keysOverviewLogsParams>;

interface ExtendedParamsKeysOverview extends KeysOverviewLogsParams {
  [key: string]: unknown;
}

export function getKeysOverviewLogs(ch: Querier) {
  return async (args: KeysOverviewLogsParams) => {
    const paramSchemaExtension: Record<string, z.ZodType> = {};
    const parameters: ExtendedParamsKeysOverview = { ...args };

    const hasKeyIdFilters = args.keyIds && args.keyIds.length > 0;
    const hasOutcomeFilters = args.outcomes && args.outcomes.length > 0;
    const hasSortingRules = args.sorts && args.sorts.length > 0;

    const outcomeCondition = !hasOutcomeFilters
      ? "TRUE"
      : args.outcomes
          ?.map((filter, index) => {
            if (filter.operator === "is") {
              const paramName = `outcomeValue_${index}`;
              paramSchemaExtension[paramName] = z.string();
              parameters[paramName] = filter.value;
              return `outcome = {${paramName}: String}`;
            }
            return null;
          })
          .filter(Boolean)
          .join(" OR ") || "TRUE";

    const keyIdConditions = !hasKeyIdFilters
      ? "TRUE"
      : args.keyIds
          ?.map((p, index) => {
            const paramName = `keyIdValue_${index}`;
            paramSchemaExtension[paramName] = z.string();
            parameters[paramName] = p.value;
            switch (p.operator) {
              case "is":
                return `key_id = {${paramName}: String}`;
              case "contains":
                return `like(key_id, CONCAT('%', {${paramName}: String}, '%'))`;
              default:
                return null;
            }
          })
          .filter(Boolean)
          .join(" OR ") || "TRUE";

    const allowedColumns = new Map([
      ["time", "last_request_time"],
      ["valid_count", "valid_count"],
      ["error_count", "error_count"],
    ]);

    const orderBy =
      hasSortingRules && args.sorts
        ? args.sorts.reduce((acc: string[], sort) => {
            const column = allowedColumns.get(sort.column);
            if (column) {
              const direction =
                sort.direction.toUpperCase() === "ASC" ||
                sort.direction.toUpperCase() === "DESC"
                  ? sort.direction.toUpperCase()
                  : "DESC";
              acc.push(`${column} ${direction}`);
            }
            return acc;
          }, [])
        : [];

    const timeSort = args.sorts?.find((s) => s.column === "time");
    const timeDirection =
      timeSort?.direction.toUpperCase() === "ASC" ? "ASC" : "DESC";

    const orderByWithoutTime = orderBy.filter(
      (clause) => !clause.startsWith("last_request_time")
    );

    const orderByClause =
      [
        ...orderByWithoutTime,
        `last_request_time ${timeDirection}`,
        `request_id ${timeDirection}`,
      ].join(", ") || "last_request_time DESC, request_id DESC";

    // Update the cursor condition to use request_id
    const cursorCondition = `
      AND (
          ({cursorTime: Nullable(UInt64)} IS NULL AND {cursorRequestId: Nullable(String)} IS NULL)
          OR (time = {cursorTime: Nullable(UInt64)} AND request_id < {cursorRequestId: Nullable(String)})
          OR time < {cursorTime: Nullable(UInt64)}
      )
    `;

    const extendedParamsSchema =
      keysOverviewLogsParams.extend(paramSchemaExtension);
    const query = ch.query({
      query: `WITH filtered_keys AS (
    SELECT
        request_id,
        time,
        key_id,
        outcome
    FROM verifications.raw_key_verifications_v1
    WHERE workspace_id = {workspaceId: String}
        AND key_space_id = {keyspaceId: String}
        AND time BETWEEN {startTime: UInt64} AND {endTime: UInt64}
        AND (${keyIdConditions})
        AND (${outcomeCondition})
        ${cursorCondition}
),
aggregated_data AS (
    SELECT 
        key_id,
        max(time) as last_request_time,
        argMax(request_id, time) as last_request_id,
        countIf(outcome = 'VALID') as valid_count,
        countIf(outcome != 'VALID') as error_count,
        groupArray((outcome, count(*))) as outcome_tuples
    FROM filtered_keys
    GROUP BY key_id
)
SELECT 
    key_id,
    last_request_time as time,
    last_request_id as request_id,
    valid_count,
    error_count,
    arrayMap(x -> (x.1, toUInt32(x.2)), outcome_tuples) as outcome_counts
FROM aggregated_data
ORDER BY ${orderByClause}
LIMIT {limit: Int}`,
      params: extendedParamsSchema,
      schema: rawKeysOverviewLogs, // Using the raw schema without key_details
    });

    // Execute the ClickHouse query
    const clickhouseResults = await query(parameters);

    if (clickhouseResults.val && clickhouseResults.val.length > 0) {
      // Transform outcome_counts array into an object
      return {
        ...clickhouseResults,
        val: clickhouseResults.val.map((result) => {
          // Convert outcome_counts from array of tuples to object if needed
          const outcomeCountsObj: Record<string, number> = {};
          if (Array.isArray(result.outcome_counts)) {
            result.outcome_counts.forEach(([outcome, count]) => {
              outcomeCountsObj[outcome] = count;
            });
            return {
              ...result,
              outcome_counts: outcomeCountsObj,
            };
          }
          return result;
        }),
      };
    }

    return clickhouseResults;
  };
}
