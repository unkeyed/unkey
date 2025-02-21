import type { TimeseriesGranularity } from "@/lib/trpc/routers/utils/granularity";
import { addMinutes, format } from "date-fns";

export const formatTimestampTooltip = (value: string | number) => {
  const date = new Date(value);
  const offset = new Date().getTimezoneOffset() * -1;
  const localDate = addMinutes(date, offset);
  return format(localDate, "MMM dd HH:mm aa");
};

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
