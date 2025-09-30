import { ControlsContainer, ControlsLeft } from "@/components/logs/controls-container";
import { DeploymentListDatetime } from "./components/deployment-list-datetime";
import { DeploymentListFilters } from "./components/deployment-list-filters";
import { DeploymentListSearch } from "./components/deployment-list-search";

export function DeploymentsListControls() {
  return (
    <ControlsContainer>
      <ControlsLeft>
        <DeploymentListSearch />
        <DeploymentListFilters />
        <DeploymentListDatetime />
      </ControlsLeft>
    </ControlsContainer>
  );
}
