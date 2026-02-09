"use client";

import { ChevronDown } from "@unkey/icons";
import { Button, Tabs, TabsList, TabsTrigger } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useDeploymentLogsContext } from "../providers/deployment-logs-provider";

export function DeploymentLogsTrigger() {
  const { isExpanded, toggleExpanded, logType, setLogType } = useDeploymentLogsContext();

  return (
    <div className="flex items-center gap-1.5">
      <Tabs value={logType} onValueChange={(val) => setLogType(val as "sentinel" | "runtime")}>
        <TabsList className="bg-gray-3 h-auto">
          <TabsTrigger value="sentinel" className="text-accent-12 text-xs px-2 py-1">
            Logs
          </TabsTrigger>
          <TabsTrigger value="runtime" className="text-accent-12 text-xs px-2 py-1">
            Runtime logs
          </TabsTrigger>
        </TabsList>
      </Tabs>
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
