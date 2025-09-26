import type { CompoundTimeseriesGranularity } from "@/lib/trpc/routers/utils/granularity";
import { format, fromUnixTime } from "date-fns";

// Memoization cache with bounded size
const formatCache = new Map<string, string>();
const MAX_CACHE_SIZE = 1000;

// Memoized format function
const memoizedFormat = (date: Date, formatString: string): string => {
  const cacheKey = `${date.getTime()}-${formatString}`;

  const cachedValue = formatCache.get(cacheKey);
  if (cachedValue !== undefined) {
    return cachedValue;
  }

  const formattedValue = format(date, formatString);

  // Evict oldest entries if cache is full
  if (formatCache.size >= MAX_CACHE_SIZE) {
    const firstKey = formatCache.keys().next().value;
    if (firstKey !== undefined) {
      formatCache.delete(firstKey);
    }
  }

  formatCache.set(cacheKey, formattedValue);
  return formattedValue;
};

export const formatTimestampLabel = (timestamp: string | number | Date) => {
  const date = new Date(timestamp);
  return memoizedFormat(date, "MMM dd, h:mm a").toUpperCase();
};

export const formatTimestampForChart = (
  value: string | number,
  granularity: CompoundTimeseriesGranularity,
) => {
  const localDate = new Date(value);

  switch (granularity) {
    case "perMinute":
      return memoizedFormat(localDate, "h:mm:ss a");
    case "per5Minutes":
    case "per15Minutes":
    case "per30Minutes":
      return memoizedFormat(localDate, "h:mm a");
    case "perHour":
    case "per2Hours":
    case "per4Hours":
    case "per6Hours":
      return memoizedFormat(localDate, "MMM d, h:mm a");
    case "perDay":
      return memoizedFormat(localDate, "MMM d");

    case "per12Hours":
      return memoizedFormat(localDate, "MMM d, h:mm a");
    case "per3Days":
      return memoizedFormat(localDate, "MMM d");
    case "perWeek":
      return memoizedFormat(localDate, "MMM d");
    case "perMonth":
      return memoizedFormat(localDate, "MMM yyyy");
    default:
      return memoizedFormat(localDate, "Pp");
  }
};

const unixMicroToDate = (unix: string | number): Date => {
  return fromUnixTime(Number(unix) / 1000 / 1000);
};

const isUnixMicro = (unix: string | number): boolean => {
  const digitLength = String(unix).length === 16;
  const isNum = !Number.isNaN(Number(unix));
  return isNum && digitLength;
};

export const formatTimestampTooltip = (value: string | number) => {
  // Coerce numeric strings to numbers to prevent Invalid Date
  const parsed = typeof value === "string" ? Number(value) : value;

  let date: Date;
  if (isUnixMicro(parsed)) {
    date = unixMicroToDate(parsed);
  } else if (typeof parsed === "number") {
    date = new Date(parsed);
  } else {
    // Fallback for non-numeric strings
    date = new Date(String(value));
  }

  return memoizedFormat(date, "MMM dd, h:mm:ss a");
};
