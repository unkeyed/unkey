import { useRequestLogsContext } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/requests/context/request-logs-provider";
import { useRequestLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/requests/hooks/use-request-logs-filters";
import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { LiveSwitchButton } from "@/components/logs/live-switch-button";

export const RequestLogsLiveSwitch = () => {
  const { isLive, toggleLive } = useRequestLogsContext();
  const { filters, updateFilters } = useRequestLogsFilters();

  const handleSwitch = () => {
    toggleLive();
    // To able to refetch historic data again we have to update the endTime
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
