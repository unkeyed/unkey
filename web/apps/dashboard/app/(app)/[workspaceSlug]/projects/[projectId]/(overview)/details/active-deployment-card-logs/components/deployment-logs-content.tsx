"use client";

import { formatNumber } from "@/lib/fmt";
import { CircleXMark, Layers3, Magnifier, TriangleWarning2 } from "@unkey/icons";
import { CopyButton, Input } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { format } from "date-fns";
import { useDeploymentLogs } from "../hooks/use-deployment-logs";
import { useDeploymentLogsContext } from "../providers/deployment-logs-provider";
import { FilterButton } from "./filter-button";
import { RuntimeLogsContent } from "./runtime-logs-content";

const ANIMATION_STYLES = {
  expand: "transition-all duration-400 ease-in",
  slideIn: "transition-all duration-500 ease-out",
} as const;

type Props = {
  projectId: string;
  deploymentId: string;
};

export function DeploymentLogsContent({ projectId, deploymentId }: Props) {
  const { isExpanded, logType } = useDeploymentLogsContext();
  const {
    logFilter,
    searchTerm,
    showFade,
    filteredLogs,
    logCounts,
    handleScroll,
    handleFilterChange,
    handleSearchChange,
    scrollRef,
  } = useDeploymentLogs({
    deploymentId,
    projectId,
  });

  if (logType === "runtime") {
    return <RuntimeLogsContent projectId={projectId} deploymentId={deploymentId} />;
  }

  return (
    <div
      className={cn(
        "bg-gray-1 relative overflow-hidden",
        ANIMATION_STYLES.expand,
        isExpanded ? "h-96 opacity-100 py-3" : "h-0 opacity-0 py-0",
      )}
    >
      <div className="flex items-center gap-1.5 px-3 mb-3">
        <FilterButton
          isActive={logFilter === "all"}
          count={formatNumber(logCounts.total)}
          onClick={() => handleFilterChange("all")}
          icon={Layers3}
          label="All Logs"
        />
        <FilterButton
          isActive={logFilter === "errors"}
          count={formatNumber(logCounts.errors)}
          onClick={() => handleFilterChange("errors")}
          icon={CircleXMark}
          label="5XX"
        />
        <FilterButton
          isActive={logFilter === "warnings"}
          count={formatNumber(logCounts.warnings)}
          onClick={() => handleFilterChange("warnings")}
          icon={TriangleWarning2}
          label="4XX"
        />

        <Input
          variant="ghost"
          wrapperClassName="ml-4"
          className="min-h-[26px] text-xs rounded-lg placeholder:text-grayA-8"
          leftIcon={<Magnifier iconSize="sm-medium" className="text-accent-9 size-[14px]!" />}
          placeholder="Find in logs..."
          value={searchTerm}
          onChange={handleSearchChange}
        />

        <CopyButton
          value={JSON.stringify(filteredLogs)}
          className="size-[22px] [&_svg]:size-[14px] ml-4"
          toastMessage="Logs copied to clipboard"
        />
      </div>

      <div
        className={cn(
          ANIMATION_STYLES.slideIn,
          "h-full",
          isExpanded ? "translate-y-0 opacity-100" : "translate-y-4 opacity-0",
        )}
        style={{
          transitionDelay: isExpanded ? "150ms" : "0ms",
        }}
      >
        <div className="h-full overflow-y-auto" onScroll={handleScroll} ref={scrollRef}>
          {filteredLogs.length === 0 ? (
            <div className="text-center text-gray-9 text-sm py-4 flex items-center justify-center h-full">
              {searchTerm
                ? `No logs match "${searchTerm}"`
                : `No ${logFilter === "all" ? "sentinel" : logFilter} logs available`}
            </div>
          ) : (
            <div className="flex flex-col gap-px">
              {filteredLogs.map((log, index) => (
                <div
                  key={`${log.message}-${index}`}
                  className={cn(
                    "font-mono text-xs flex gap-6 items-center text-[11px] leading-7 font-medium",
                    "transition-all duration-300 ease-out text-grayA-12 ",
                    isExpanded ? "translate-x-0 opacity-100" : "translate-x-2 opacity-0",
                    log.level === "warning"
                      ? "bg-linear-to-r from-warningA-3 to-warningA-1 text-warningA-12"
                      : log.level === "error"
                        ? "bg-linear-to-r from-errorA-3 to-errorA-1 text-errorA-12"
                        : "",
                  )}
                  style={{
                    transitionDelay: isExpanded ? `${200 + index * 20}ms` : "0ms",
                  }}
                >
                  <span className="text-grayA-9 pl-3">
                    {format(new Date(log.timestamp), "HH:mm:ss.SSS")}
                  </span>
                  <span className="pr-3">{log.message}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Fade overlay */}
      {showFade && (
        <div className="absolute bottom-0 left-0 right-0 h-8 bg-linear-to-t from-gray-1 to-transparent pointer-events-none transition-opacity duration-300 ease-out" />
      )}
    </div>
  );
}
