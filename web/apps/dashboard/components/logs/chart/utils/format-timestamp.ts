import type { TimeseriesGranularity } from "@/lib/trpc/routers/utils/granularity";
import { parseTimestamp } from "@unkey/ui";
import { format } from "date-fns";

// Memoization cache with bounded size
// Speed improvement so we do not repeat each timeStampFormat
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
  const isNumericString = typeof timestamp === "string" && /^\d+$/.test(timestamp);
  const input = isNumericString ? Number(timestamp) : timestamp;
  const date = input instanceof Date ? input : new Date(input);
  return memoizedFormat(date, "MMM dd, h:mm a").toUpperCase();
};

export const formatTimestampForChart = (
  value: string | number,
  granularity: TimeseriesGranularity,
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
    case "perQuarter":
      return memoizedFormat(localDate, "MMM yyyy");
    default:
      return memoizedFormat(localDate, "Pp");
  }
};

export const formatTimestampTooltip = (value: string | number | Date) => {
  const date = parseTimestamp(value);
  if (Number.isNaN(date.getTime())) {
    return String(value);
  }
  return memoizedFormat(date, "MMM dd, h:mm:ss a");
};
