import { ControlsContainer, ControlsLeft } from "@/components/logs/controls-container";
import { IdentitiesSearch } from "./identities-search";

export function IdentitiesListControls() {
  return (
    <ControlsContainer>
      <ControlsLeft>
        <IdentitiesSearch />
      </ControlsLeft>
    </ControlsContainer>
  );
}
