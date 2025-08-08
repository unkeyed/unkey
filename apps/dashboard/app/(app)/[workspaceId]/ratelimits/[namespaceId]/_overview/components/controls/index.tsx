import {
  ControlsContainer,
  ControlsLeft,
  ControlsRight,
} from "@/components/logs/controls-container";
import { LogsDateTime } from "./components/logs-datetime";
import { LogsFilters } from "./components/logs-filters";
import { LogsRefresh } from "./components/logs-refresh";
import { LogsSearch } from "./components/logs-search";

export function RatelimitOverviewLogsControls() {
  return (
    <ControlsContainer>
      <ControlsLeft>
        <LogsSearch />
        <LogsFilters />
        <LogsDateTime />
      </ControlsLeft>
      <ControlsRight>
        <LogsRefresh />
      </ControlsRight>
    </ControlsContainer>
  );
}
