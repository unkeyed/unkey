import {
  ControlsContainer,
  ControlsLeft,
  ControlsRight,
} from "@/components/logs/controls-container";
import { SentinelLogsDateTime } from "./components/sentinel-logs-datetime";
import { SentinelLogsFilters } from "./components/sentinel-logs-filters";
import { SentinelLogsRefresh } from "./components/sentinel-logs-refresh";
import { SentinelLogsSearch } from "./components/sentinel-logs-search";

export function SentinelLogsControls() {
  return (
    <ControlsContainer>
      <ControlsLeft>
        <SentinelLogsSearch />
        <SentinelLogsFilters />
        <SentinelLogsDateTime />
      </ControlsLeft>
      <ControlsRight>
        <SentinelLogsRefresh />
      </ControlsRight>
    </ControlsContainer>
  );
}
