import { useMemo } from "react";
import type { FilterValue } from "../filters.type";

export const useTimeRange = (filters: FilterValue[]) => {
  return useMemo(() => {
    const startTimeFilter = filters.find((f) => f.field === "startTime");
    const endTimeFilter = filters.find((f) => f.field === "endTime");
    const sinceFilter = filters.find((f) => f.field === "since");

    // If since is present, it takes precedence over startTime
    if (sinceFilter) {
      return {
        startTime: getTimestampFromRelative(sinceFilter.value as string),
        endTime: (endTimeFilter?.value as number | undefined) || Date.now(),
      };
    }

    return {
      startTime: startTimeFilter?.value as number | undefined,
      endTime: endTimeFilter?.value as number | undefined,
    };
  }, [filters]);
};

export const getTimestampFromRelative = (relativeTime: string): number => {
  let totalMilliseconds = 0;

  for (const [, amount, unit] of relativeTime.matchAll(/(\d+)([hdm])/g)) {
    const value = Number.parseInt(amount, 10);

    switch (unit) {
      case "h":
        totalMilliseconds += value * 60 * 60 * 1000;
        break;
      case "d":
        totalMilliseconds += value * 24 * 60 * 60 * 1000;
        break;
      case "m":
        totalMilliseconds += value * 60 * 1000;
        break;
    }
  }

  return Date.now() - totalMilliseconds;
};
