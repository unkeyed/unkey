"use client";

import { formatNumber } from "@/lib/fmt";
import { CircleXMark, Layers3, Magnifier, TriangleWarning2 } from "@unkey/icons";
import { CopyButton, InfoTooltip, Input } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { format } from "date-fns";
import { useRuntimeLogs } from "../hooks/use-runtime-logs";
import { useDeploymentLogsContext } from "../providers/deployment-logs-provider";
import { FilterButton } from "./filter-button";

const ANIMATION_STYLES = {
  expand: "transition-all duration-400 ease-in",
  slideIn: "transition-all duration-500 ease-out",
} as const;

const SEVERITY_STYLES = {
  ERROR: "bg-gradient-to-r from-errorA-3 to-errorA-1 text-errorA-12",
  WARN: "bg-gradient-to-r from-warningA-3 to-warningA-1 text-warningA-12",
  INFO: "text-grayA-12",
  DEBUG: "text-grayA-9",
} as const;

type Props = {
  projectId: string;
  deploymentId: string;
};

export function RuntimeLogsContent({ projectId, deploymentId }: Props) {
  const { isExpanded } = useDeploymentLogsContext();
  const {
    logFilter,
    searchTerm,
    showFade,
    filteredLogs,
    logCounts,
    isLoading,
    handleScroll,
    handleFilterChange,
    handleSearchChange,
    scrollRef,
  } = useRuntimeLogs({ projectId, deploymentId });

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
          label="Errors"
        />
        <FilterButton
          isActive={logFilter === "warnings"}
          count={formatNumber(logCounts.warnings)}
          onClick={() => handleFilterChange("warnings")}
          icon={TriangleWarning2}
          label="Warnings"
        />

        <Input
          variant="ghost"
          wrapperClassName="ml-4"
          className="min-h-[26px] text-xs rounded-lg placeholder:text-grayA-8"
          leftIcon={<Magnifier iconSize="sm-medium" className="text-accent-9 !size-[14px]" />}
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
          {isLoading && filteredLogs.length === 0 ? (
            <div className="text-center text-gray-9 text-sm py-4 flex items-center justify-center h-full">
              Loading logs...
            </div>
          ) : filteredLogs.length === 0 ? (
            <div className="text-center text-gray-9 text-sm py-4 flex items-center justify-center h-full">
              {searchTerm
                ? `No logs match "${searchTerm}"`
                : `No ${logFilter === "all" ? "runtime" : logFilter} logs available`}
            </div>
          ) : (
            <div className="flex flex-col gap-px">
              {filteredLogs.map((log, index) => {
                const severity = log.severity.toUpperCase() as keyof typeof SEVERITY_STYLES;
                const severityStyle = SEVERITY_STYLES[severity] ?? SEVERITY_STYLES.INFO;

                return (
                  <div
                    key={`${log.deployment_id}-${log.time}-${index}`}
                    className={cn(
                      "font-mono text-xs flex gap-6 items-center text-[11px] leading-7 font-medium",
                      "transition-all duration-300 ease-out",
                      isExpanded ? "translate-x-0 opacity-100" : "translate-x-2 opacity-0",
                      severityStyle,
                    )}
                    style={{
                      transitionDelay: isExpanded ? `${200 + index * 20}ms` : "0ms",
                    }}
                  >
                    <span className="text-grayA-9 pl-3">
                      {format(new Date(log.time), "HH:mm:ss.SSS")}
                    </span>
                    <span>{log.region}</span>
                    <span>{log.message}</span>
                    <InfoTooltip
                      content={
                        <pre className="text-left">
                          <code>{JSON.stringify(log.attributes, null, 2)}</code>
                        </pre>
                      }
                      asChild
                      className="cursor-pointer"
                      position={{
                        align: "center",
                        side: "top",
                      }}
                    >
                      <span className="text-grayA-8 text-[11px] max-w-[200px] truncate">
                        {JSON.stringify(log.attributes)}
                      </span>
                    </InfoTooltip>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>

      {showFade && (
        <div className="absolute bottom-0 left-0 right-0 h-8 bg-gradient-to-t from-gray-1 to-transparent pointer-events-none transition-opacity duration-300 ease-out" />
      )}
    </div>
  );
}
