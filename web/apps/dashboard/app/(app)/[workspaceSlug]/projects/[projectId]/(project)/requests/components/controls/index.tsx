import {
  ControlsContainer,
  ControlsLeft,
  ControlsRight,
} from "@/components/logs/controls-container";
import { RequestLogsDateTime } from "./components/request-logs-datetime";
import { RequestLogsFilters } from "./components/request-logs-filters";
import { RequestLogsLiveSwitch } from "./components/request-logs-live-switch";
import { RequestLogsRefresh } from "./components/request-logs-refresh";
import { RequestLogsSearch } from "./components/request-logs-search";

export function RequestLogsControls() {
  return (
    <ControlsContainer>
      <ControlsLeft>
        <RequestLogsSearch />
        <RequestLogsFilters />
        <RequestLogsDateTime />
      </ControlsLeft>
      <ControlsRight>
        <RequestLogsLiveSwitch />
        <RequestLogsRefresh />
      </ControlsRight>
    </ControlsContainer>
  );
}
