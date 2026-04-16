import { DeploymentListDatetime } from "./components/deployment-list-datetime";
import { DeploymentListFilters } from "./components/deployment-list-filters";
import { DeploymentListSearch } from "./components/deployment-list-search";

export function DeploymentsListControls() {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <div className="border border-grayA-4 rounded-lg overflow-hidden flex-1 [&_div.group>div]:!h-9 [&_input]:!font-normal">
        <DeploymentListSearch />
      </div>
      <DeploymentListFilters />
      <DeploymentListDatetime />
    </div>
  );
}
