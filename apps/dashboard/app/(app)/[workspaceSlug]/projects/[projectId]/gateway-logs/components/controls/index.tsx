import {
  ControlsContainer,
  ControlsLeft,
  ControlsRight,
} from "@/components/logs/controls-container";
import { GatewayLogsDateTime } from "./components/gateway-logs-datetime";
import { GatewayLogsFilters } from "./components/gateway-logs-filters";
import { GatewayLogsLiveSwitch } from "./components/gateway-logs-live-switch";
import { GatewayLogsRefresh } from "./components/gateway-logs-refresh";
import { GatewayLogsSearch } from "./components/gateway-logs-search";

export function GatewayLogsControls() {
  return (
    <ControlsContainer>
      <ControlsLeft>
        <GatewayLogsSearch />
        <GatewayLogsFilters />
        <GatewayLogsDateTime />
      </ControlsLeft>
      <ControlsRight>
        <GatewayLogsLiveSwitch />
        <GatewayLogsRefresh />
      </ControlsRight>
    </ControlsContainer>
  );
}
