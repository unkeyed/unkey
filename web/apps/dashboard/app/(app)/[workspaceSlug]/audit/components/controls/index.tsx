import {
  ControlsContainer,
  ControlsLeft,
  ControlsRight,
} from "@/components/logs/controls-container";
import { TimeRangeFilter } from "@/components/logs/datetime/time-range-filter";
import { Separator } from "@unkey/ui";
import { useFilters } from "../../hooks/use-filters";
import type { WorkspaceProps } from "../logs-client";
import { LogsFilters } from "./components/logs-filters";
import { LogsQueries } from "./components/logs-queries";
import { LogsRefresh } from "./components/logs-refresh";
import { LogsSearch } from "./components/logs-search";

export function AuditLogsControls(props: WorkspaceProps) {
  const { filters, updateFilters } = useFilters();
  return (
    <ControlsContainer>
      <ControlsLeft>
        <LogsSearch />
        <LogsFilters {...props} />
        <TimeRangeFilter filters={filters} updateFilters={updateFilters} look="logs" />
        <Separator
          orientation="vertical"
          className="flex items-center justify-center h-4 mx-1 my-auto"
        />
        <LogsQueries />
      </ControlsLeft>
      <ControlsRight>
        <LogsRefresh />
      </ControlsRight>
    </ControlsContainer>
  );
}
