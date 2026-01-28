"use client";

import { ChevronDown } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useDeploymentLogsContext } from "../providers/deployment-logs-provider";

type Props = {
  showBuildSteps: boolean;
};

export function DeploymentLogsTrigger({ showBuildSteps }: Props) {
  const { isExpanded, toggleExpanded } = useDeploymentLogsContext();

  return (
    <button className="flex items-center gap-1.5" onClick={toggleExpanded} type="button">
      <div className="text-grayA-9 text-xs">{showBuildSteps ? "Build logs" : "Sentinel logs"}</div>
      <Button size="icon" variant="ghost">
        <ChevronDown
          className={cn(
            "text-grayA-9 !size-3 transition-transform duration-200",
            isExpanded && "rotate-180",
          )}
        />
      </Button>
    </button>
  );
}
