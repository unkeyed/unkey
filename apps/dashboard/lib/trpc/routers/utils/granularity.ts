import {
  DAY_IN_MS,
  HOUR_IN_MS,
  MINUTE_IN_MS,
  WEEK_IN_MS,
  MONTH_IN_MS,
  QUARTER_IN_MS,
} from "./constants";

export type TimeseriesGranularity =
  | "perMinute"
  | "per5Minutes"
  | "per15Minutes"
  | "per30Minutes"
  | "perHour"
  | "per2Hours"
  | "per4Hours"
  | "per6Hours"
  | "per12Hours"
  | "perDay"
  | "per3Days"
  | "perWeek"
  | "perMonth"
  | "perQuarter";

export type TimeseriesContext = "forVerifications" | "forRegular";

const DEFAULT_GRANULARITY: Record<TimeseriesContext, TimeseriesGranularity> = {
  forVerifications: "perHour",
  forRegular: "perMinute",
};

export type TimeseriesConfig<TContext extends TimeseriesContext> = {
  granularity: TimeseriesGranularity;
  startTime: number;
  endTime: number;
  context: TContext;
};

export const TIMESERIES_GRANULARITIES = {
  perMinute: { ms: MINUTE_IN_MS, format: "h:mm:ssa" },
  per5Minutes: { ms: 5 * MINUTE_IN_MS, format: "h:mma" },
  per15Minutes: { ms: 15 * MINUTE_IN_MS, format: "h:mma" },
  per30Minutes: { ms: 30 * MINUTE_IN_MS, format: "h:mma" },
  perHour: { ms: HOUR_IN_MS, format: "MMM d h:mma" },
  per2Hours: { ms: 2 * HOUR_IN_MS, format: "MMM d h:mma" },
  per4Hours: { ms: 4 * HOUR_IN_MS, format: "MMM d h:mma" },
  per6Hours: { ms: 6 * HOUR_IN_MS, format: "MMM d h:mma" },
  per12Hours: { ms: 12 * HOUR_IN_MS, format: "MMM d h:mma" },
  perDay: { ms: DAY_IN_MS, format: "MMM d" },
  per3Days: { ms: 3 * DAY_IN_MS, format: "MMM d" },
  perWeek: { ms: WEEK_IN_MS, format: "MMM d" },
  perMonth: { ms: MONTH_IN_MS, format: "MMM yy" },
  perQuarter: { ms: QUARTER_IN_MS, format: "MMM yy" },
} as const;

/**
 * Returns an appropriate timeseries configuration based on the time range and context
 * @param context The context for which to get a granularity ("forVerifications", "forRegular")
 * @param startTime Optional start time in milliseconds
 * @param endTime Optional end time in milliseconds
 * @returns TimeseriesConfig with the appropriate granularity for the given context and time range
 */
export const getTimeseriesGranularity = <TContext extends TimeseriesContext>(
  context: TContext,
  startTime?: number | null,
  endTime?: number | null
): TimeseriesConfig<TContext> => {
  const now = Date.now();

  // If both are missing, fallback to an appropriate default for the context
  if (!startTime && !endTime) {
    const defaultGranularity = DEFAULT_GRANULARITY[context];
    const defaultDuration =
      context === "forVerifications" ? DAY_IN_MS : HOUR_IN_MS;
    return {
      granularity: defaultGranularity as TimeseriesGranularity,
      startTime: now - defaultDuration,
      endTime: now,
      context,
    };
  }

  // Set default end time if missing
  const effectiveEndTime = endTime ?? now;
  // Set default start time if missing (defaults vary by context)
  const defaultDuration =
    context === "forVerifications" ? DAY_IN_MS : HOUR_IN_MS;
  const effectiveStartTime = startTime ?? effectiveEndTime - defaultDuration;
  const timeRange = effectiveEndTime - effectiveStartTime;
  let granularity: TimeseriesGranularity;

  if (timeRange >= QUARTER_IN_MS) {
    granularity = "perMonth";
  } else if (timeRange >= MONTH_IN_MS * 2) {
    granularity = "perWeek";
  } else if (timeRange >= MONTH_IN_MS) {
    granularity = "per3Days";
  } else if (timeRange >= WEEK_IN_MS) {
    granularity = "perDay";
  } else if (timeRange >= DAY_IN_MS * 3) {
    granularity = "perHour";
  } else if (timeRange >= DAY_IN_MS) {
    granularity = "perHour";
  } else if (timeRange >= HOUR_IN_MS * 16) {
    granularity = "perHour";
  } else if (timeRange >= HOUR_IN_MS * 8) {
    granularity = "per30Minutes";
  } else if (timeRange >= HOUR_IN_MS * 4) {
    granularity = "per5Minutes";
  } else {
    granularity = "perMinute";
  }

  return {
    granularity: granularity as TimeseriesGranularity,
    startTime: effectiveStartTime,
    endTime: effectiveEndTime,
    context,
  };
};

/**
 * Returns the appropriate buffer in milliseconds for a given granularity
 * Use this to expand time ranges when exact timestamps are selected
 * @param granularity The current timeseries granularity
 * @returns Buffer time in milliseconds
 */
export const getTimeBufferForGranularity = (
  granularity: TimeseriesGranularity
): number => {
  // Constants for commonly used durations

  // Return appropriate buffer based on granularity
  switch (granularity) {
    case "perMinute":
      return MINUTE_IN_MS;
    case "per5Minutes":
      return 5 * MINUTE_IN_MS;
    case "per15Minutes":
      return 15 * MINUTE_IN_MS;
    case "per30Minutes":
      return 30 * MINUTE_IN_MS;
    case "perHour":
      return HOUR_IN_MS;
    case "per2Hours":
      return 2 * HOUR_IN_MS;
    case "per4Hours":
      return 4 * HOUR_IN_MS;
    case "per6Hours":
      return 6 * HOUR_IN_MS;
    case "per12Hours":
      return 12 * HOUR_IN_MS;
    case "perDay":
      return DAY_IN_MS;
    case "per3Days":
      return 3 * DAY_IN_MS;
    case "perWeek":
      return WEEK_IN_MS;
    case "perMonth":
      return MONTH_IN_MS;
    case "perQuarter":
      return QUARTER_IN_MS;
    default:
      // Default to 5 minutes if granularity is unknown
      return 5 * MINUTE_IN_MS;
  }
};
