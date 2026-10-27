import {
  ControlsContainer,
  ControlsLeft,
  ControlsRight,
} from "@/components/logs/controls-container";
import { TimeRangeFilter } from "@/components/time-range-filter";
import { useFilters } from "../../hooks/use-filters";
import { LogsFilters } from "./components/logs-filters";
import { LogsRefresh } from "./components/logs-refresh";
import { LogsSearch } from "./components/logs-search";

export function RatelimitOverviewLogsControls() {
  const { filters, updateFilters } = useFilters();
  return (
    <ControlsContainer>
      <ControlsLeft>
        <LogsSearch />
        <LogsFilters />
        <TimeRangeFilter filters={filters} updateFilters={updateFilters} look="logs" />
      </ControlsLeft>
      <ControlsRight>
        <LogsRefresh />
      </ControlsRight>
    </ControlsContainer>
  );
}
