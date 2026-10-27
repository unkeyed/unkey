import { DEFAULT_OPTIONS } from "@/components/logs/datetime/constants";
import { useWorkspaceLimits } from "@/hooks/use-workspace-limits";
import { getTimestampFromRelative } from "@/lib/duration";
import { DAY_IN_MS } from "@/lib/trpc/routers/utils/constants";
import { useMemo } from "react";

export function useRetentionTimeRange() {
  const { data: limits } = useWorkspaceLimits();
  const retentionDays = limits?.logsRetentionDaysMax.limit ?? null;

  const presetsWithinRetention = useMemo(() => {
    if (retentionDays === null) {
      return DEFAULT_OPTIONS;
    }
    const now = Date.now();
    return DEFAULT_OPTIONS.filter(
      (option) =>
        option.value === undefined ||
        now - getTimestampFromRelative(option.value, now) <= retentionDays * DAY_IN_MS,
    );
  }, [retentionDays]);

  return {
    key: retentionDays ?? "unlimited",
    customOptions: presetsWithinRetention,
    minDate: retentionDays === null ? undefined : new Date(Date.now() - retentionDays * DAY_IN_MS),
  };
}
