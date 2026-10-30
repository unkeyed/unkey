import {
  ControlsContainer,
  ControlsLeft,
  ControlsRight,
} from "@/components/logs/controls-container";
import { TimeRangeFilter } from "@/components/logs/datetime/time-range-filter";
import { Separator } from "@unkey/ui";
import { useFilters } from "../../hooks/use-filters";
import { useRetentionTimeRange } from "../../hooks/use-retention-time-range";
import { LogsDisplay } from "./components/logs-display";
import { LogsFilters } from "./components/logs-filters";
import { LogsLiveSwitch } from "./components/logs-live-switch";
import { LogsQueries } from "./components/logs-queries";
import { LogsRefresh } from "./components/logs-refresh";
import { LogsSearch } from "./components/logs-search";

export function LogsControls() {
  const { filters, updateFilters } = useFilters();
  const retention = useRetentionTimeRange();
  return (
    <ControlsContainer>
      <ControlsLeft>
        <LogsSearch />
        <LogsFilters />
        <TimeRangeFilter
          filters={filters}
          updateFilters={updateFilters}
          look="logs"
          {...retention}
        />
        <Separator
          orientation="vertical"
          className="flex items-center justify-center h-4 mx-1 my-auto"
        />
        <LogsQueries />
      </ControlsLeft>
      <ControlsRight>
        <LogsLiveSwitch />
        <LogsRefresh />
        <LogsDisplay />
      </ControlsRight>
    </ControlsContainer>
  );
}
