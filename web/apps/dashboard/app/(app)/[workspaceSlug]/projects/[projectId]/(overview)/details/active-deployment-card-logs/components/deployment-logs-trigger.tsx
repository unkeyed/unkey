"use client";

import { ChevronDown } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useDeploymentLogsContext } from "../providers/deployment-logs-provider";

export function DeploymentLogsTrigger() {
  const { isExpanded, toggleExpanded, logType, setLogType } = useDeploymentLogsContext();

  return (
    <div className="flex items-center gap-1.5">
      <button
        onClick={() => setLogType("sentinel")}
        className={cn("text-xs", logType === "sentinel" ? "text-accent-12" : "text-grayA-9")}
        type="button"
      >
        Sentinel logs
      </button>
      <span className="text-grayA-6">|</span>
      <button
        onClick={() => setLogType("runtime")}
        className={cn("text-xs", logType === "runtime" ? "text-accent-12" : "text-grayA-9")}
        type="button"
      >
        Runtime logs
      </button>
      <Button size="icon" variant="ghost" onClick={toggleExpanded}>
        <ChevronDown
          className={cn(
            "text-grayA-9 !size-3 transition-transform duration-200",
            isExpanded && "rotate-180",
          )}
        />
      </Button>
    </div>
  );
}
