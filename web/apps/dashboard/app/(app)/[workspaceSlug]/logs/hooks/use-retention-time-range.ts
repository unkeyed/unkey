import { presetsWithinRetention } from "@/components/logs/datetime/constants";
import { useWorkspaceLimits } from "@/hooks/use-workspace-limits";
import { DAY_IN_MS } from "@/lib/trpc/routers/utils/constants";
import { useMemo } from "react";

export function useRetentionTimeRange() {
  const { data: limits } = useWorkspaceLimits();
  const retentionDays = limits?.logsRetentionDaysMax.limit ?? null;
  const customOptions = useMemo(() => presetsWithinRetention(retentionDays), [retentionDays]);

  return {
    customOptions,
    minDate: retentionDays === null ? undefined : new Date(Date.now() - retentionDays * DAY_IN_MS),
  };
}
