import { ControlsContainer, ControlsLeft } from "@/components/logs/controls-container";
import { DeploymentListFilters } from "./components/deployment-list-filters";
import { DeploymentListSearch } from "./components/deployment-list-search";

export function DeploymentsListControls() {
  return (
    <ControlsContainer>
      <ControlsLeft>
        <DeploymentListSearch />
        <DeploymentListFilters />
      </ControlsLeft>
    </ControlsContainer>
  );
}
