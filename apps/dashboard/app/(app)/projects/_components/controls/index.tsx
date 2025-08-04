import {
  ControlsContainer,
  ControlsLeft,
  ControlsRight,
} from "@/components/logs/controls-container";
import { LogsDisplay } from "./components/logs-display";
import { LogsFilters } from "./components/logs-filters";
import { LogsSearch } from "./components/logs-search";

export function LogsControls() {
  return (
    <ControlsContainer>
      <ControlsLeft>
        <LogsSearch />
        <LogsFilters />
      </ControlsLeft>
      <ControlsRight>
        <LogsDisplay />
      </ControlsRight>
    </ControlsContainer>
  );
}
