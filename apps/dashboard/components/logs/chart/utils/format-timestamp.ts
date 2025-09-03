import type { CompoundTimeseriesGranularity } from "@/lib/trpc/routers/utils/granularity";
import { format, fromUnixTime } from "date-fns";

export const formatTimestampLabel = (timestamp: string | number | Date) => {
  const date = new Date(timestamp);
  return format(date, "MMM dd, h:mm a").toUpperCase();
};

export const formatTimestampForChart = (
  value: string | number,
  granularity: CompoundTimeseriesGranularity,
) => {
  const localDate = new Date(value);

  switch (granularity) {
    case "perMinute":
      return format(localDate, "h:mm:ss a");
    case "per5Minutes":
    case "per15Minutes":
    case "per30Minutes":
      return format(localDate, "h:mm a");
    case "perHour":
    case "per2Hours":
    case "per4Hours":
    case "per6Hours":
      return format(localDate, "MMM d, h:mm a");
    case "perDay":
      return format(localDate, "MMM d");

    case "per12Hours":
      return format(localDate, "MMM d, h:mm a");
    case "per3Days":
      return format(localDate, "MMM d");
    case "perWeek":
      return format(localDate, "MMM d");
    case "perMonth":
      return format(localDate, "MMM yyyy");
    default:
      return format(localDate, "Pp");
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
  const date = isUnixMicro(value) ? unixMicroToDate(value) : new Date(value);
  return format(date, "MMM dd h:mm:ss a");
};
