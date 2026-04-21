import { z } from "zod";
import type { Querier } from "../client";

export const KEY_VERIFICATION_OUTCOMES = [
  "VALID",
  "INSUFFICIENT_PERMISSIONS",
  "RATE_LIMITED",
  "FORBIDDEN",
  "DISABLED",
  "EXPIRED",
  "USAGE_EXCEEDED",
  "",
] as const;
export const keysOverviewLogsParams = z.object({
  workspaceId: z.string(),
  keyspaceId: z.string(),
  limit: z.int(),
  startTime: z.int(),
  endTime: z.int(),
  outcomes: z
    .array(
      z.object({
        value: z.enum(KEY_VERIFICATION_OUTCOMES),
        operator: z.literal("is"),
      }),
    )
    .nullable(),
  names: z
    .array(
      z.object({
        operator: z.enum(["is", "contains", "startsWith", "endsWith"]),
        value: z.string(),
      }),
    )
    .nullable(),
  identities: z
    .array(
      z.object({
        operator: z.enum(["is", "contains", "startsWith", "endsWith"]),
        value: z.string(),
      }),
    )
    .nullable(),
  keyIds: z
    .array(
      z.object({
        operator: z.enum(["is", "contains"]),
        value: z.string(),
      }),
    )
    .nullable(),
  tags: z
    .array(
      z.object({
        operator: z.enum(["is", "contains", "startsWith", "endsWith"]),
        value: z.string(),
      }),
    )
    .nullable(),
  cursorTime: z.int().nullable(),
  page: z.int().min(1).optional(),
  sorts: z
    .array(
      z.object({
        column: z.enum(["time", "valid", "invalid"]),
        direction: z.enum(["asc", "desc"]),
      }),
    )
    .nullable(),
  useTimeFrameFilter: z.boolean().optional(),
});

export const roleSchema = z.object({
  name: z.string(),
  description: z.string().nullable(),
});

export type Role = z.infer<typeof roleSchema>;

export const permissionSchema = z.object({
  name: z.string(),
  description: z.string().nullable(),
});

export type Permission = z.infer<typeof permissionSchema>;

export const identitySchema = z.object({
  external_id: z.string().nullable(),
});

export type Identity = z.infer<typeof identitySchema>;

export const keyDetailsResponseSchema = z.object({
  id: z.string(),
  key_auth_id: z.string(),
  name: z.string().nullable(),
  owner_id: z.string().nullable(),
  identity_id: z.string().nullable(),
  meta: z.string().nullable(),
  enabled: z.boolean(),
  remaining_requests: z.number().nullable(),
  environment: z.string().nullable(),
  workspace_id: z.string(),
  identity: identitySchema.nullable(),
  roles: z.array(roleSchema),
  permissions: z.array(permissionSchema),
});
export type KeyDetailsResponse = z.infer<typeof keyDetailsResponseSchema>;

export const rawKeysOverviewLogs = z.object({
  time: z.int(),
  key_id: z.string(),
  request_id: z.string(),
  valid_count: z.int(),
  error_count: z.int(),
  outcome_counts: z.record(z.string(), z.int()),
  tags: z.array(z.string()).optional(),
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
    const hasTagFilters = args.tags && args.tags.length > 0;
    const hasSortingRules = args.sorts && args.sorts.length > 0;

    const outcomeCondition = hasOutcomeFilters
      ? args.outcomes
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
          .join(" OR ") || "TRUE"
      : "TRUE";

    const keyIdConditions = hasKeyIdFilters
      ? args.keyIds
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
          .join(" OR ") || "TRUE"
      : "TRUE";

    const tagConditions = hasTagFilters
      ? args.tags
          ?.map((filter, index) => {
            const paramName = `tagValue_${index}`;
            paramSchemaExtension[paramName] = z.string();
            parameters[paramName] = filter.value;
            switch (filter.operator) {
              case "is":
                return `has(tags, {${paramName}: String})`;
              case "contains":
                return `arrayExists(x -> like(x, CONCAT('%', {${paramName}: String}, '%')), tags)`;
              case "startsWith":
                return `arrayExists(x -> like(x, CONCAT({${paramName}: String}, '%')), tags)`;
              case "endsWith":
                return `arrayExists(x -> like(x, CONCAT('%', {${paramName}: String})), tags)`;
              default:
                return null;
            }
          })
          .filter(Boolean)
          .join(" OR ") || "TRUE"
      : "TRUE";

    // Map user-facing sort columns to the aggregated column names exposed by
    // the top_keys CTE. Unknown columns are dropped to prevent SQL injection.
    const allowedColumns = new Map([
      ["time", "time"],
      ["valid", "valid_count"],
      ["invalid", "error_count"],
    ]);

    const userOrderBy = hasSortingRules
      ? (args.sorts ?? []).flatMap((sort) => {
          const column = allowedColumns.get(sort.column);
          if (!column) {
            return [];
          }
          const direction = sort.direction.toUpperCase() === "ASC" ? "ASC" : "DESC";
          return [`${column} ${direction}`];
        })
      : [];

    // Time is always the final tiebreaker so results are deterministic across
    // pages, even when the primary sort column has ties.
    const hasExplicitTimeSort = userOrderBy.some((clause) => clause.startsWith("time "));
    const orderByClause = hasExplicitTimeSort
      ? userOrderBy.join(", ")
      : [...userOrderBy, "time DESC"].join(", ");

    // Page-based pagination: translate page → OFFSET. page is 1-indexed.
    const page = args.page ?? 1;
    const offset = (page - 1) * args.limit;
    parameters.offset = offset;
    paramSchemaExtension.offset = z.int();

    const extendedParamsSchema = keysOverviewLogsParams.extend(paramSchemaExtension);

    // Single-pass query (3 staged CTEs, one scan of hourly + raw):
    //   1. per_key_outcome — read hourly + raw once, aggregate per
    //      (key_id, outcome). argMax uses event_time to deterministically
    //      pick the latest request_id / tags for each (key, outcome) pair.
    //   2. key_aggregates — roll up per key. valid_count / error_count drive
    //      sorting, and groupArray builds the per-outcome tuple for the UI's
    //      InvalidCount breakdown without needing a second scan.
    //   3. Outer SELECT — add `count() OVER ()` for pagination, order, and
    //      limit. Kept in its own stage (separate from GROUP BY) because CH
    //      has had quirks combining `count() OVER ()` with aggregation + LIMIT
    //      in the same SELECT, occasionally dropping rows with tied sort keys.
    const query = ch.query({
      query: `
WITH per_key_outcome AS (
    SELECT
        key_id,
        outcome,
        sum(event_count) as outcome_count,
        max(event_time) as last_time,
        argMax(event_request_id, event_time) as last_request_id,
        argMax(event_tags, event_time) as last_tags
    FROM (
        SELECT
            key_id,
            outcome,
            toInt64(toUnixTimestamp(time) * 1000) as event_time,
            '' as event_request_id,
            tags as event_tags,
            count as event_count
        FROM default.key_verifications_per_hour_v3
        WHERE workspace_id = {workspaceId: String}
            AND key_space_id = {keyspaceId: String}
            AND time BETWEEN toDateTime(fromUnixTimestamp64Milli({startTime: UInt64}))
                         AND toDateTime(fromUnixTimestamp64Milli({endTime: UInt64}))
            AND time < toStartOfHour(now())
            AND (${keyIdConditions})
            AND (${outcomeCondition})
            AND (${tagConditions})

        UNION ALL

        SELECT
            key_id,
            outcome,
            time as event_time,
            request_id as event_request_id,
            tags as event_tags,
            1 as event_count
        FROM default.key_verifications_raw_v2
        WHERE workspace_id = {workspaceId: String}
            AND key_space_id = {keyspaceId: String}
            AND time >= toUnixTimestamp(toStartOfHour(now())) * 1000
            AND time BETWEEN {startTime: UInt64} AND {endTime: UInt64}
            AND (${keyIdConditions})
            AND (${outcomeCondition})
            AND (${tagConditions})
    )
    GROUP BY key_id, outcome
),
key_aggregates AS (
    SELECT
        key_id,
        max(last_time) as time,
        argMax(last_request_id, last_time) as request_id,
        argMax(last_tags, last_time) as tags,
        sumIf(outcome_count, outcome = 'VALID') as valid_count,
        sumIf(outcome_count, outcome != 'VALID') as error_count,
        arrayFilter(x -> tupleElement(x, 2) > 0,
            groupArray(tuple(outcome, toUInt32(outcome_count)))
        ) as outcome_counts_array
    FROM per_key_outcome
    GROUP BY key_id
    HAVING valid_count > 0 OR error_count > 0
)
SELECT
    key_id,
    time,
    request_id,
    tags,
    valid_count,
    error_count,
    outcome_counts_array,
    count() OVER () as total_count
FROM key_aggregates
ORDER BY ${orderByClause}, key_id ASC
LIMIT {limit: Int}
OFFSET {offset: Int}
`,
      params: extendedParamsSchema,
      schema: rawKeysOverviewLogs
        .extend({
          outcome_counts_array: z.array(z.tuple([z.string(), z.number()])),
          total_count: z.int(),
        })
        .omit({ outcome_counts: true }),
    });

    const sharedResult = query(parameters);

    const logsQuery = (async () => {
      const result = await sharedResult;
      if (result.err) {
        return result;
      }
      return {
        err: result.err,
        val: (result.val ?? []).map((row) => {
          const outcomeCountsObj: Record<string, number> = {};
          if (Array.isArray(row.outcome_counts_array)) {
            for (const [outcome, count] of row.outcome_counts_array) {
              outcomeCountsObj[outcome] = count;
            }
          }
          return {
            key_id: row.key_id,
            time: row.time,
            request_id: row.request_id,
            tags: row.tags,
            valid_count: row.valid_count,
            error_count: row.error_count,
            outcome_counts: outcomeCountsObj,
          };
        }),
      };
    })();

    // Total count rides on every row (window function); extract it from the
    // first row, or 0 if no keys matched. Keeps the same {logsQuery, countQuery}
    // interface the tRPC router expects.
    const countQuery = (async () => {
      const result = await sharedResult;
      if (result.err) {
        return { err: result.err, val: undefined };
      }
      const total = result.val?.[0]?.total_count ?? 0;
      return { err: undefined, val: [{ total_count: total }] };
    })();

    return {
      logsQuery,
      countQuery,
    };
  };
}
