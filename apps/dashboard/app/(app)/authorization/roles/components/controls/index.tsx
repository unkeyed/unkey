import { ControlsContainer, ControlsLeft } from "@/components/logs/controls-container";
import { LogsFilters } from "./components/logs-filters";
import { LogsSearch } from "./components/logs-search";

export function RoleListControls() {
  return (
    <ControlsContainer>
      <ControlsLeft>
        <LogsSearch keyspaceId={""} />
        <LogsFilters />
      </ControlsLeft>
    </ControlsContainer>
  );
}
