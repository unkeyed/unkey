import {
  ControlsContainer,
  ControlsLeft,
  ControlsRight,
} from "@/components/logs/controls-container";
import { TimeRangeFilter } from "@/components/time-range-filter";
import { useRequestLogsFilters } from "../../hooks/use-request-logs-filters";
import { RequestLogsFilters } from "./components/request-logs-filters";
import { RequestLogsLiveSwitch } from "./components/request-logs-live-switch";
import { RequestLogsRefresh } from "./components/request-logs-refresh";
import { RequestLogsSearch } from "./components/request-logs-search";

export function RequestLogsControls() {
  const { filters, updateFilters } = useRequestLogsFilters();
  return (
    <ControlsContainer>
      <ControlsLeft>
        <RequestLogsSearch />
        <RequestLogsFilters />
        <TimeRangeFilter
          filters={filters}
          updateFilters={updateFilters}
          look="logs"
          defaultTitle="Last 6 hours"
        />
      </ControlsLeft>
      <ControlsRight>
        <RequestLogsLiveSwitch />
        <RequestLogsRefresh />
      </ControlsRight>
    </ControlsContainer>
  );
}
