import {
  ControlsContainer,
  ControlsLeft,
  ControlsRight,
} from "@/components/logs/controls-container";
import { LogsDateTime } from "./components/logs-datetime";
import { LogsFilters } from "./components/logs-filters";
import { LogsRefresh } from "./components/logs-refresh";
import { LogsSearch } from "./components/logs-search";

export function KeysDetailsLogsControls({
  keyspaceId,
  keyId,
}: {
  keyId: string;
  keyspaceId: string;
}) {
  return (
    <ControlsContainer>
      <ControlsLeft>
        <LogsSearch keyspaceId={keyspaceId} keyId={keyId} />
        <LogsFilters />
        <LogsDateTime />
      </ControlsLeft>
      <ControlsRight>
        {/* <LogsLiveSwitch /> */}
        <LogsRefresh />
      </ControlsRight>
    </ControlsContainer>
  );
}
