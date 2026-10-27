import {
  ControlsContainer,
  ControlsLeft,
  ControlsRight,
} from "@/components/logs/controls-container";
import { HOUR_PLUS_OPTIONS } from "@/components/logs/datetime/constants";
import { TimeRangeFilter } from "@/components/time-range-filter";
import { useFilters } from "../../hooks/use-filters";
import { LogsFilters } from "./components/logs-filters";
import { LogsRefresh } from "./components/logs-refresh";
import { LogsSearch } from "./components/logs-search";

export function KeysOverviewLogsControls({ apiId }: { apiId: string }) {
  const { filters, updateFilters } = useFilters();
  return (
    <ControlsContainer>
      <ControlsLeft>
        <LogsSearch apiId={apiId} />
        <LogsFilters />
        <TimeRangeFilter
          filters={filters}
          updateFilters={updateFilters}
          look="logsCompact"
          customOptions={HOUR_PLUS_OPTIONS}
        />
      </ControlsLeft>
      <ControlsRight>
        <LogsRefresh />
      </ControlsRight>
    </ControlsContainer>
  );
}
