import type { TimeseriesGranularity } from "@/lib/trpc/routers/utils/granularity";
import { addMinutes, format, fromUnixTime } from "date-fns";

export const formatTimestampLabel = (timestamp: string | number | Date) => {
  const date = new Date(timestamp);
  return format(date, "MMM dd, h:mma").toUpperCase();
};

export const formatTimestampForChart = (
  value: string | number,
  granularity: TimeseriesGranularity,
) => {
  const date = new Date(value);
  const offset = new Date().getTimezoneOffset() * -1;
  const localDate = addMinutes(date, offset);

  switch (granularity) {
    case "perMinute":
      return format(localDate, "HH:mm:ss");
    case "per5Minutes":
    case "per15Minutes":
    case "per30Minutes":
      return format(localDate, "HH:mm");
    case "perHour":
    case "per2Hours":
    case "per4Hours":
    case "per6Hours":
      return format(localDate, "MMM d, HH:mm");
    case "perDay":
      return format(localDate, "MMM d");
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
  return format(date, "MMM dd HH:mm:ss");
};
