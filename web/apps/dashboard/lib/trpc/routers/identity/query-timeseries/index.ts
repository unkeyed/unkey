import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { getTimestampFromRelative } from "@/lib/utils";
import { TRPCError } from "@trpc/server";
import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import type { VerificationTimeseriesDataPoint } from "@unkey/clickhouse/src/verifications";
import type { getMinutelyIdentityTimeseries } from "@unkey/clickhouse/src/verifications";
import { z } from "zod";
import {
  type TimeseriesConfig,
  type TimeseriesGranularity,
  getTimeseriesGranularity,
} from "../../utils/granularity";

// Input schema for identity timeseries query
export const identityTimeseriesPayload = z.object({
  identityId: z.string(),
  startTime: z.number().int(),
  endTime: z.number().int(),
  granularity: z.enum([
    "minute",
    "fiveMinutes",
    "fifteenMinutes",
    "thirtyMinutes",
    "hour",
    "twoHours",
    "fourHours",
    "sixHours",
    "twelveHours",
    "day",
    "threeDays",
    "week",
    "month",
    "quarter",
  ]),
  since: z.string().optional(),
  tags: z
    .array(
      z.object({
        value: z.string(),
        operator: z.enum(["is", "contains", "startsWith", "endsWith"]),
      }),
    )
    .optional()
    .nullable(),
  outcomes: z
    .array(
      z.object({
        value: z.enum(KEY_VERIFICATION_OUTCOMES),
        operator: z.literal("is"),
      }),
    )
    .optional()
    .nullable(),
});

const identityTimeseriesResponse = z.object({
  timeseries: z.array(z.custom<VerificationTimeseriesDataPoint>()),
  granularity: z.enum([
    "perMinute",
    "per5Minutes",
    "per15Minutes",
    "per30Minutes",
    "perHour",
    "per2Hours",
    "per4Hours",
    "per6Hours",
    "per12Hours",
    "perDay",
    "per3Days",
    "perWeek",
    "perMonth",
    "perQuarter",
  ]),
});

type IdentityTimeseriesResponse = z.infer<typeof identityTimeseriesResponse>;

export const queryIdentityTimeseries = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(identityTimeseriesPayload)
  .output(identityTimeseriesResponse)
  .query(async ({ ctx, input }) => {
    // First, validate identity exists and get associated keys
    const identity = await db.query.identities
      .findFirst({
        where: (table, { eq }) => eq(table.id, input.identityId),
        with: {
          workspace: {
            columns: { id: true, orgId: true },
          },
          keys: {
            where: (keysTable, { isNull }) => isNull(keysTable.deletedAtM),
            with: {
              keyAuth: {
                columns: { id: true },
              },
            },
            columns: { id: true, keyAuthId: true },
          },
        },
      })
      .catch((err) => {
        console.error("Failed to retrieve identity details:", err);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve identity details due to an error. If this issue persists, please contact support@unkey.dev with the time this occurred.",
        });
      });

    if (!identity) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Identity not found, please contact support using support@unkey.dev.",
      });
    }

    if (identity.workspace.orgId !== ctx.tenant.id) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Identity not found in the specified workspace.",
      });
    }

    if (!identity.keys || identity.keys.length === 0) {
      // Return empty timeseries if identity has no keys
      return {
        timeseries: [],
        granularity: "perHour" as const, // Default granularity in correct format
      };
    }

    // Handle time conversion - similar to transformVerificationFilters
    let timeConfig: TimeseriesConfig<"forVerifications">;

    if (input.since && input.since !== "") {
      const startTime = getTimestampFromRelative(input.since);
      const endTime = Date.now();
      timeConfig = getTimeseriesGranularity("forVerifications", startTime, endTime);
    } else {
      timeConfig = getTimeseriesGranularity("forVerifications", input.startTime, input.endTime);
    }

    // Prepare parameters for identity timeseries query
    const keyIds = identity.keys.map((key) => key.id);

    const timeseriesParams = {
      workspaceId: identity.workspace.id,
      keyIds, // Array of key IDs for identity aggregation across keyspaces
      startTime: timeConfig.startTime,
      endTime: timeConfig.endTime,
      tags: input.tags || null,
      outcomes: input.outcomes || null,
    };

    // Query ClickHouse using identity timeseries functions
    let result: Awaited<ReturnType<ReturnType<typeof getMinutelyIdentityTimeseries>>>;
    try {
      // Map TimeseriesGranularity back to input granularity for the switch statement
      const granularityMap: Record<TimeseriesGranularity, string> = {
        perMinute: "minute",
        per5Minutes: "fiveMinutes",
        per15Minutes: "fifteenMinutes",
        per30Minutes: "thirtyMinutes",
        perHour: "hour",
        per2Hours: "twoHours",
        per4Hours: "fourHours",
        per6Hours: "sixHours",
        per12Hours: "twelveHours",
        perDay: "day",
        per3Days: "threeDays",
        perWeek: "week",
        perMonth: "month",
        perQuarter: "quarter",
      };

      const mappedGranularity = granularityMap[timeConfig.granularity];

      // Use the new identity timeseries functions that don't require keyspaceId
      switch (mappedGranularity) {
        case "minute":
          result = await clickhouse.api.identity.timeseries.perMinute(timeseriesParams);
          break;
        case "fiveMinutes":
          result = await clickhouse.api.identity.timeseries.per5Minutes(timeseriesParams);
          break;
        case "fifteenMinutes":
          result = await clickhouse.api.identity.timeseries.per15Minutes(timeseriesParams);
          break;
        case "thirtyMinutes":
          result = await clickhouse.api.identity.timeseries.per30Minutes(timeseriesParams);
          break;
        case "hour":
          result = await clickhouse.api.identity.timeseries.perHour(timeseriesParams);
          break;
        case "twoHours":
          result = await clickhouse.api.identity.timeseries.per2Hours(timeseriesParams);
          break;
        case "fourHours":
          result = await clickhouse.api.identity.timeseries.per4Hours(timeseriesParams);
          break;
        case "sixHours":
          result = await clickhouse.api.identity.timeseries.per6Hours(timeseriesParams);
          break;
        case "twelveHours":
          result = await clickhouse.api.identity.timeseries.per12Hours(timeseriesParams);
          break;
        case "day":
          result = await clickhouse.api.identity.timeseries.perDay(timeseriesParams);
          break;
        case "threeDays":
          result = await clickhouse.api.identity.timeseries.per3Days(timeseriesParams);
          break;
        case "week":
          result = await clickhouse.api.identity.timeseries.perWeek(timeseriesParams);
          break;
        case "month":
          result = await clickhouse.api.identity.timeseries.perMonth(timeseriesParams);
          break;
        case "quarter":
          result = await clickhouse.api.identity.timeseries.perQuarter(timeseriesParams);
          break;
        default:
          throw new TRPCError({
            code: "BAD_REQUEST",
            message: `Unsupported granularity: ${mappedGranularity}`,
          });
      }
    } catch (error) {
      console.error("ClickHouse identity timeseries query failed:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching identity timeseries data from ClickHouse.",
      });
    }

    const timeseries = result?.val || [];

    const response: IdentityTimeseriesResponse = {
      timeseries,
      granularity: timeConfig.granularity,
    };

    return response;
  });
