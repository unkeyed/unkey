import { ControlsContainer, ControlsLeft } from "@/components/logs/controls-container";
import { LogsFilters } from "./components/logs-filters";
import { PermissionSearch } from "./components/logs-search";

export function PermissionListControls() {
  return (
    <ControlsContainer>
      <ControlsLeft>
        <PermissionSearch />
        <LogsFilters />
      </ControlsLeft>
    </ControlsContainer>
  );
}
