import {
  ControlsContainer,
  ControlsLeft,
  ControlsRight,
} from "@/components/logs/controls-container";
import { LogsDateTime } from "./components/logs-datetime";
import { LogsFilters } from "./components/logs-filters";
import { LogsRefresh } from "./components/logs-refresh";
import { LogsSearch } from "./components/logs-search";

export function KeysOverviewLogsControls({ apiId }: { apiId: string }) {
  return (
    <ControlsContainer>
      <ControlsLeft>
        <LogsSearch apiId={apiId} />
        <LogsFilters />
        <LogsDateTime />
      </ControlsLeft>
      <ControlsRight>
        <LogsRefresh />
      </ControlsRight>
    </ControlsContainer>
  );
}
