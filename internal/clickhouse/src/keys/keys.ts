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
  limit: z.number().int(),
  startTime: z.number().int(),
  endTime: z.number().int(),
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
  cursorTime: z.number().int().nullable(),
  sorts: z
    .array(
      z.object({
        column: z.enum(["time", "valid", "invalid"]),
        direction: z.enum(["asc", "desc"]),
      }),
    )
    .nullable(),
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
  time: z.number().int(),
  key_id: z.string(),
  request_id: z.string(),
  valid_count: z.number().int(),
  error_count: z.number().int(),
  outcome_counts: z.record(z.string(), z.number().int()),
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
    let cursorCondition: string;

    // For first page or no cursor provided
    if (args.cursorTime) {
      // For subsequent pages, use cursor based on time direction
      if (timeDirection === "ASC") {
        cursorCondition = `
        AND (time > {cursorTime: Nullable(UInt64)})
        `;
      } else {
        cursorCondition = `
        AND (time < {cursorTime: Nullable(UInt64)})
        `;
      }
    } else {
      cursorCondition = `
      AND ({cursorTime: Nullable(UInt64)} IS NULL)
      `;
    }

    const extendedParamsSchema = keysOverviewLogsParams.extend(paramSchemaExtension);
    const query = ch.query({
      query: `
WITH
    -- First CTE: Filter raw verification records based on conditions from client
    filtered_keys AS (
      SELECT
          request_id,
          time,
          key_id,
          tags,
          outcome
      FROM verifications.raw_key_verifications_v1
      WHERE workspace_id = {workspaceId: String}
          AND key_space_id = {keyspaceId: String}
          AND time BETWEEN {startTime: UInt64} AND {endTime: UInt64}
          -- Apply dynamic key ID filtering (equals or contains)
          AND (${keyIdConditions})
          -- Apply dynamic outcome filtering
          AND (${outcomeCondition})
          -- Apply dynamic tag filtering
          AND (${tagConditions})
          -- Handle pagination using only time as cursor
          ${cursorCondition}
    ),
    -- Second CTE: Calculate per-key aggregated metrics
    -- This groups all verifications by key_id to get summary counts and most recent activity
    aggregated_data AS (
      SELECT
          key_id,
          -- Find the timestamp of the latest verification for this key
          max(time) as last_request_time,
          -- Get the request_id of the latest verification (based on time)
          argMax(request_id, time) as last_request_id,
          -- Get the tags from the latest verification (based on time)
          argMax(tags, time) as tags,
          -- Count valid verifications
          countIf(outcome = 'VALID') as valid_count,
          -- Count all non-valid verifications
          countIf(outcome != 'VALID') as error_count
      FROM filtered_keys
      GROUP BY key_id
    ),
    -- Third CTE: Build detailed outcome distribution
    -- This provides a breakdown of the exact counts for each outcome type
    outcome_counts AS (
      SELECT
          key_id,
          outcome,
          -- Convert to UInt32 for consistency
          toUInt32(count(*)) as count
      FROM filtered_keys
      GROUP BY key_id, outcome
    )
    -- Main query: Join the aggregated data with detailed outcome counts
    SELECT
      a.key_id,
      a.last_request_time as time,
      a.last_request_id as request_id,
      a.tags,
      a.valid_count,
      a.error_count,
      -- Create an array of tuples containing all outcomes and their counts
      -- This will be transformed into an object in the application code
      groupArray((o.outcome, o.count)) as outcome_counts_array
    FROM aggregated_data a
    LEFT JOIN outcome_counts o ON a.key_id = o.key_id
    -- Group by all non-aggregated fields to allow the groupArray operation
    GROUP BY
      a.key_id,
      a.last_request_time,
      a.last_request_id,
      a.tags,
      a.valid_count,
      a.error_count
    -- Sort results with most recent verification first
    ORDER BY ${orderByClause}
    -- Limit results for pagination
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
