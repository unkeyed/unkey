"use client";

import { useRuntimeLogs } from "@/app/(app)/[workspaceSlug]/projects/[projectSlug]/(project)/logs/context/runtime-logs-provider";
import { useRuntimeLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectSlug]/(project)/logs/hooks/use-runtime-logs-filters";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { LiveSwitchButton } from "@/components/logs/live-switch-button";

export const RuntimeLogsLiveSwitch = () => {
  const { isLive, toggleLive } = useRuntimeLogs();
  const { filters, updateFilters } = useRuntimeLogsFilters();

  const handleSwitch = () => {
    toggleLive();
    // Re-anchor the time window when leaving Live so historical fetches
    // resume from "now" instead of a stale endTime.
    if (isLive) {
      const timestamp = Date.now();
      const activeFilters = filters.filter((f) => !["endTime", "startTime"].includes(f.field));
      updateFilters([
        ...activeFilters,
        {
          field: "endTime",
          value: timestamp,
          id: crypto.randomUUID(),
          operator: "is",
        },
        {
          field: "startTime",
          value: timestamp - HISTORICAL_DATA_WINDOW,
          id: crypto.randomUUID(),
          operator: "is",
        },
      ]);
    }
  };

  return <LiveSwitchButton onToggle={handleSwitch} isLive={isLive} />;
};
