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

    const allowedColumns = new Map([
      ["time", "time"],
      ["valid", "valid_count"],
      ["invalid", "error_count"],
    ]);

    const orderBy =
      hasSortingRules && args.sorts
        ? args.sorts.reduce((acc: string[], sort) => {
            const column = allowedColumns.get(sort.column);
            // Only add to ORDER BY if it's an allowed column to prevent injection
            if (column) {
              const direction =
                sort.direction.toUpperCase() === "ASC" || sort.direction.toUpperCase() === "DESC"
                  ? sort.direction.toUpperCase()
                  : "DESC";
              acc.push(`${column} ${direction}`);
            }
            return acc;
          }, [])
        : [];

    // Check if we have sorts for valid or invalid
    const hasValidSort = args.sorts?.some((s) => s.column === "valid");
    const hasInvalidSort = args.sorts?.some((s) => s.column === "invalid");
    const hasCustomSort = hasValidSort || hasInvalidSort;

    // Get explicit time sort if it exists
    const timeSort = args.sorts?.find((s) => s.column === "time");

    // Determine time direction:
    // If we have custom sort (valid/invalid), always use ASC for better pagination
    // Otherwise use explicit time direction or default to DESC
    const timeDirection = hasCustomSort
      ? "ASC"
      : timeSort?.direction.toUpperCase() === "ASC"
        ? "ASC"
        : "DESC";

    // Remove any existing time sort from the orderBy array
    const orderByWithoutTime = orderBy.filter((clause) => !clause.startsWith("time"));

    // Construct final ORDER BY clause with only time at the end
    const orderByClause =
      [...orderByWithoutTime, `time ${timeDirection}`].join(", ") || "time DESC"; // Fallback if empty

    // Create cursor condition based on time direction
    let havingCursorCondition: string;

    if (args.cursorTime) {
      // For subsequent pages, use cursor based on time direction
      if (timeDirection === "ASC") {
        havingCursorCondition = "\n          AND (last_time > {cursorTime: Nullable(UInt64)})";
      } else {
        havingCursorCondition = "\n          AND (last_time < {cursorTime: Nullable(UInt64)})";
      }
    } else {
      havingCursorCondition = "";
    }

    // Detect if this is a rolling/relative time window (last X hours/days)
    // vs an explicit historical range
    const now = Date.now();
    // If user explicitly filtered by time, use historical path to find ALL keys with activity in window
    // Otherwise use MV fast path for recent "last used" data
    const isRollingWindow = !args.useTimeFrameFilter && args.endTime >= now - 5 * 60 * 1000;

    const extendedParamsSchema = keysOverviewLogsParams.extend(paramSchemaExtension);

    // Build top_keys CTE based on query type
    const topKeysCTE = isRollingWindow
      ? `top_keys AS (
      SELECT
          key_id,
          workspace_id,
          key_space_id,
          max(time) as last_time,
          anyLast(request_id) as last_request_id,
          anyLast(tags) as last_tags
      FROM default.key_last_used_v1
      WHERE workspace_id = {workspaceId: String}
          AND key_space_id = {keyspaceId: String}
          AND (${keyIdConditions})
          AND time >= {startTime: UInt64}
          AND time BETWEEN {startTime: UInt64} AND {endTime: UInt64}
      GROUP BY key_id, workspace_id, key_space_id
      HAVING last_time > 0${havingCursorCondition}
      ORDER BY last_time ${timeDirection}
      LIMIT {limit: Int}
    )`
      : `top_keys AS (
      SELECT
          key_id,
          workspace_id,
          key_space_id,
          max(time) as last_time,
          anyLast(request_id) as last_request_id,
          anyLast(tags) as last_tags
      FROM (
          -- Get activity from hourly aggregates (complete hours)
          SELECT
              key_id,
              workspace_id,
              key_space_id,
              toInt64(toUnixTimestamp(time) * 1000) as time,
              '' as request_id,
              tags
          FROM default.key_verifications_per_hour_v2
          WHERE workspace_id = {workspaceId: String}
              AND key_space_id = {keyspaceId: String}
              AND time BETWEEN toDateTime(fromUnixTimestamp64Milli({startTime: UInt64}))
                           AND toDateTime(fromUnixTimestamp64Milli({endTime: UInt64}))
              AND (${keyIdConditions})
          
          UNION ALL
          
          -- Get activity from raw table (current incomplete hour)
          SELECT
              key_id,
              workspace_id,
              key_space_id,
              time,
              request_id,
              tags
          FROM default.key_verifications_raw_v2
          WHERE workspace_id = {workspaceId: String}
              AND key_space_id = {keyspaceId: String}
              AND time BETWEEN {startTime: UInt64} AND {endTime: UInt64}
              AND (${keyIdConditions})
      )
      GROUP BY key_id, workspace_id, key_space_id
      HAVING last_time > 0
          ${havingCursorCondition}
      ORDER BY last_time ${timeDirection}
      LIMIT {limit: Int}
    )`;

    const query = ch.query({
      query: `
WITH
    ${topKeysCTE},
    -- Second CTE: Get counts from hourly table (complete hours only)
    hourly_counts AS (
      SELECT
          h.key_id,
          h.outcome,
          toUInt64(sum(h.count)) as count
      FROM default.key_verifications_per_hour_v2 h
      INNER JOIN top_keys t ON h.key_id = t.key_id
      WHERE h.workspace_id = {workspaceId: String}
          AND h.key_space_id = {keyspaceId: String}
          AND h.time BETWEEN toDateTime(fromUnixTimestamp64Milli({startTime: UInt64})) 
                         AND toDateTime(fromUnixTimestamp64Milli({endTime: UInt64}))
          AND h.time < toStartOfHour(now())  -- Only complete hours
          AND (${outcomeCondition})
          AND (${tagConditions})
      GROUP BY h.key_id, h.outcome
    ),
    -- Third CTE: Get counts from raw table for current incomplete hour
    recent_counts AS (
      SELECT
          v.key_id,
          v.outcome,
          toUInt64(count(*)) as count
      FROM default.key_verifications_raw_v2 v
      INNER JOIN top_keys t ON v.key_id = t.key_id
      WHERE v.workspace_id = {workspaceId: String}
          AND v.key_space_id = {keyspaceId: String}
          AND v.time >= toUnixTimestamp(toStartOfHour(now())) * 1000
          AND v.time BETWEEN {startTime: UInt64} AND {endTime: UInt64}
          AND (${outcomeCondition})
          AND (${tagConditions})
      GROUP BY v.key_id, v.outcome
    ),
    -- Fourth CTE: Combine hourly and recent counts
    combined_counts AS (
      SELECT key_id, outcome, count FROM hourly_counts
      UNION ALL
      SELECT key_id, outcome, count FROM recent_counts
    ),
    -- Fifth CTE: Aggregate combined counts
    aggregated_counts AS (
      SELECT
          key_id,
          sumIf(count, outcome = 'VALID') as valid_count,
          sumIf(count, outcome != 'VALID') as error_count
      FROM combined_counts
      GROUP BY key_id
    ),
    -- Sixth CTE: Build outcome distribution
    outcome_counts AS (
      SELECT
          key_id,
          outcome,
          toUInt32(sum(count)) as count
      FROM combined_counts
      GROUP BY key_id, outcome
    )
    -- Main query: Join metadata from MV with aggregated counts
    SELECT
      t.key_id as key_id,
      t.last_time as time,
      t.last_request_id as request_id,
      t.last_tags as tags,
      COALESCE(a.valid_count, 0) as valid_count,
      COALESCE(a.error_count, 0) as error_count,
      arrayFilter(x -> tupleElement(x, 1) IS NOT NULL, 
        groupArray(tuple(o.outcome, o.count))
      ) as outcome_counts_array
    FROM top_keys t
    LEFT JOIN aggregated_counts a ON t.key_id = a.key_id
    LEFT JOIN outcome_counts o ON t.key_id = o.key_id
    GROUP BY
      t.key_id,
      t.last_time,
      t.last_request_id,
      t.last_tags,
      a.valid_count,
      a.error_count
    HAVING COALESCE(a.valid_count, 0) > 0 OR COALESCE(a.error_count, 0) > 0
    ORDER BY ${orderByClause}
    LIMIT {limit: Int}
`,
      params: extendedParamsSchema,
      schema: rawKeysOverviewLogs
        .extend({
          outcome_counts_array: z.array(z.tuple([z.string(), z.number()])),
        })
        .omit({ outcome_counts: true }),
    });

    // Execute the ClickHouse query
    const clickhouseResults = await query(parameters);

    // Always transform the results to ensure consistent structure
    return {
      ...clickhouseResults,
      val: (clickhouseResults.val || []).map((result) => {
        // Convert outcome_counts_array from array of tuples to object
        const outcomeCountsObj: Record<string, number> = {};
        if (Array.isArray(result.outcome_counts_array)) {
          result.outcome_counts_array.forEach(([outcome, count]) => {
            outcomeCountsObj[outcome] = count;
          });
        }

        // Create a new object with the correct structure
        return {
          key_id: result.key_id,
          time: result.time,
          request_id: result.request_id,
          tags: result.tags,
          valid_count: result.valid_count,
          error_count: result.error_count,
          outcome_counts: outcomeCountsObj,
        };
      }),
    };
  };
}
