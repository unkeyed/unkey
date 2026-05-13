import { BranchSelect } from "./components/branch-select";
import { DeploymentListDatetime } from "./components/deployment-list-datetime";
import { EnvironmentSelect } from "./components/environment-select";
import { StatusSelect } from "./components/status-select";

export function DeploymentsListControls() {
  return (
    <div className="flex flex-col md:flex-row items-stretch gap-2">
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
    </div>
  );
}
