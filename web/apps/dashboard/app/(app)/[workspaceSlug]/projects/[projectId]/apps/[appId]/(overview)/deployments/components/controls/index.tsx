import { ResourceListHeader } from "@unkey/ui";
import { BranchSelect } from "./components/branch-select";
import { DeploymentListDatetime } from "./components/deployment-list-datetime";
import { EnvironmentSelect } from "./components/environment-select";
import { StatusSelect } from "./components/status-select";

export function DeploymentsListControls() {
  return (
    <ResourceListHeader>
      <div className="w-full md:flex-1">
        <EnvironmentSelect />
      </div>
      <div className="w-full md:flex-1">
        <StatusSelect />
      </div>
      <div className="w-full md:flex-1">
        <BranchSelect />
      </div>
      <div className="w-full md:flex-1">
        <DeploymentListDatetime />
      </div>
    </ResourceListHeader>
  );
}
