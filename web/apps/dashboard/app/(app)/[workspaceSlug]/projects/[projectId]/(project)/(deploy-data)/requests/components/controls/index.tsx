import {
  ControlsContainer,
  ControlsLeft,
  ControlsRight,
} from "@/components/logs/controls-container";
import { TimeRangeFilter } from "@/components/logs/datetime/time-range-filter";
import { DEFAULT_LOGS_SINCE } from "@/lib/utils";
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
          defaultSince={DEFAULT_LOGS_SINCE}
        />
      </ControlsLeft>
      <ControlsRight>
        <RequestLogsLiveSwitch />
        <RequestLogsRefresh />
      </ControlsRight>
    </ControlsContainer>
  );
}
