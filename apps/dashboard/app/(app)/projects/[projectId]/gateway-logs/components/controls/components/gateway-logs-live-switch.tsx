import { HISTORICAL_DATA_WINDOW } from "@/components/logs/constants";
import { LiveSwitchButton } from "@/components/logs/live-switch-button";
import { useGatewayLogsContext } from "../../../context/gateway-logs-provider";
import { useGatewayLogsFilters } from "../../../hooks/use-gateway-logs-filters";

export const GatewayLogsLiveSwitch = () => {
  const { toggleLive, isLive } = useGatewayLogsContext();
  const { filters, updateFilters } = useGatewayLogsFilters();

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
