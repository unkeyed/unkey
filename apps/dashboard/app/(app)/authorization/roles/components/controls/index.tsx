import { ControlsContainer, ControlsLeft } from "@/components/logs/controls-container";
import { LogsFilters } from "./components/logs-filters";
import { RolesSearch } from "./components/logs-search";

export function RoleListControls() {
  return (
    <ControlsContainer>
      <ControlsLeft>
        <RolesSearch />
        <LogsFilters />
      </ControlsLeft>
    </ControlsContainer>
  );
}
