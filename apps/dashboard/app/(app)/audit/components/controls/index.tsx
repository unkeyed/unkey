import { ControlsContainer, ControlsLeft, ControlsRight } from "@/components/logs/logs-container";
import type { WorkspaceProps } from "../logs-client";
import { LogsDateTime } from "./components/logs-datetime";
import { LogsFilters } from "./components/logs-filters";
import { LogsRefresh } from "./components/logs-refresh";
import { LogsSearch } from "./components/logs-search";
import { Separator } from "@/components/ui/separator";
import { LogsQueries } from "./components/logs-queries";

export function AuditLogsControls(props: WorkspaceProps) {
  return (
    <ControlsContainer>
      <ControlsLeft>
        <LogsSearch />
        <LogsFilters {...props} />
        <LogsDateTime />
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
