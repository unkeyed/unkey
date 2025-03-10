import { DAY_IN_MS, HOUR_IN_MS } from "./constants";

export type VerificationTimeseriesGranularity =
  | "perDay"
  | "per12Hours"
  | "per3Days"
  | "perWeek"
  | "perMonth"
  | "perHour";

export type RegularTimeseriesGranularity =
  | "perMinute"
  | "per5Minutes"
  | "per15Minutes"
  | "per30Minutes"
  | "perHour"
  | "per2Hours"
  | "per4Hours"
  | "per6Hours";

export type TimeseriesContext = "forVerifications" | "forRegular";

export type TimeseriesGranularityMap = {
  forVerifications: VerificationTimeseriesGranularity;
  forRegular: RegularTimeseriesGranularity;
};

export type CompoundTimeseriesGranularity =
  | VerificationTimeseriesGranularity
  | RegularTimeseriesGranularity;

const DEFAULT_GRANULARITY: Record<TimeseriesContext, CompoundTimeseriesGranularity> = {
  forVerifications: "perHour",
  forRegular: "perMinute",
};

export type TimeseriesConfig<TContext extends TimeseriesContext> = {
  granularity: TimeseriesGranularityMap[TContext];
  startTime: number;
  endTime: number;
  context: TContext;
};

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
  endTime?: number | null,
): TimeseriesConfig<TContext> => {
  const now = Date.now();
  const WEEK_IN_MS = DAY_IN_MS * 7;
  const MONTH_IN_MS = DAY_IN_MS * 30;
  const QUARTER_IN_MS = MONTH_IN_MS * 3;

  // If both are missing, fallback to an appropriate default for the context
  if (!startTime && !endTime) {
    const defaultGranularity = DEFAULT_GRANULARITY[context];
    const defaultDuration = context === "forVerifications" ? DAY_IN_MS : HOUR_IN_MS;

    return {
      granularity: defaultGranularity as TimeseriesGranularityMap[TContext],
      startTime: now - defaultDuration,
      endTime: now,
      context,
    };
  }

  // Set default end time if missing
  const effectiveEndTime = endTime ?? now;
  // Set default start time if missing (defaults vary by context)
  const defaultDuration = context === "forVerifications" ? DAY_IN_MS : HOUR_IN_MS;
  const effectiveStartTime = startTime ?? effectiveEndTime - defaultDuration;

  const timeRange = effectiveEndTime - effectiveStartTime;

  let granularity: CompoundTimeseriesGranularity;

  if (context === "forVerifications") {
    if (timeRange >= QUARTER_IN_MS) {
      granularity = "perMonth";
    } else if (timeRange >= MONTH_IN_MS * 2) {
      granularity = "perWeek";
    } else if (timeRange >= MONTH_IN_MS) {
      granularity = "per3Days";
    } else if (timeRange >= WEEK_IN_MS * 2) {
      granularity = "perDay";
    } else if (timeRange >= WEEK_IN_MS) {
      granularity = "per12Hours";
    } else {
      granularity = "perHour";
    }
  } else {
    if (timeRange >= DAY_IN_MS * 7) {
      granularity = "perDay";
    } else if (timeRange >= DAY_IN_MS * 3) {
      granularity = "per6Hours";
    } else if (timeRange >= HOUR_IN_MS * 24) {
      granularity = "per4Hours";
    } else if (timeRange >= HOUR_IN_MS * 16) {
      granularity = "per2Hours";
    } else if (timeRange >= HOUR_IN_MS * 12) {
      granularity = "perHour";
    } else if (timeRange >= HOUR_IN_MS * 8) {
      granularity = "per30Minutes";
    } else if (timeRange >= HOUR_IN_MS * 6) {
      granularity = "per30Minutes";
    } else if (timeRange >= HOUR_IN_MS * 4) {
      granularity = "per15Minutes";
    } else if (timeRange >= HOUR_IN_MS * 2) {
      granularity = "per5Minutes";
    } else {
      granularity = "perMinute";
    }
  }

  return {
    granularity: granularity as TimeseriesGranularityMap[TContext],
    startTime: effectiveStartTime,
    endTime: effectiveEndTime,
    context,
  };
};
