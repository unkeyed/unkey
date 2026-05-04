import { ControlsContainer, ControlsLeft } from "@/components/logs/controls-container";
import { IdentitiesDateTime } from "./components/identities-datetime";
import { IdentitiesFilters } from "./components/identities-filters";
import { IdentitiesSearch } from "./identities-search";

export function IdentitiesListControls() {
  return (
    <ControlsContainer>
      <ControlsLeft>
        <IdentitiesSearch />
        <IdentitiesFilters />
        <IdentitiesDateTime />
      </ControlsLeft>
    </ControlsContainer>
  );
}
