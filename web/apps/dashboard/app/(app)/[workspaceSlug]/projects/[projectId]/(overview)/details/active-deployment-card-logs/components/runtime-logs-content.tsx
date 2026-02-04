"use client";

import { TimestampInfo } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useRuntimeLogs } from "../hooks/use-runtime-logs";
import { useDeploymentLogsContext } from "../providers/deployment-logs-provider";

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

const SEVERITY_ABBR = {
  ERROR: "ERR",
  WARN: "WRN",
  INFO: "INF",
  DEBUG: "DBG",
} as const;

type Props = {
  projectId: string;
  deploymentId: string;
};

export function RuntimeLogsContent({ projectId, deploymentId }: Props) {
  const { isExpanded } = useDeploymentLogsContext();
  const { logs, isLoading } = useRuntimeLogs({ projectId, deploymentId });

  return (
    <div
      className={cn(
        "bg-gray-1 relative overflow-hidden",
        ANIMATION_STYLES.expand,
        isExpanded ? "h-96 opacity-100 py-3" : "h-0 opacity-0 py-0",
      )}
    >
      <div
        className={cn(
          ANIMATION_STYLES.slideIn,
          "h-full px-3",
          isExpanded ? "translate-y-0 opacity-100" : "translate-y-4 opacity-0",
        )}
        style={{
          transitionDelay: isExpanded ? "150ms" : "0ms",
        }}
      >
        <div className="h-full overflow-y-auto">
          {isLoading && logs.length === 0 ? (
            <div className="text-center text-gray-9 text-sm py-4 flex items-center justify-center h-full">
              Loading runtime logs...
            </div>
          ) : logs.length === 0 ? (
            <div className="text-center text-gray-9 text-sm py-4 flex items-center justify-center h-full">
              No runtime logs available
            </div>
          ) : (
            <div className="flex flex-col gap-px">
              {logs.map((log, index) => {
                const severity = log.severity.toUpperCase() as keyof typeof SEVERITY_STYLES;
                const severityStyle = SEVERITY_STYLES[severity] ?? SEVERITY_STYLES.INFO;

                return (
                  <div
                    key={`${log.deployment_id}-${log.time}-${index}`}
                    className={cn(
                      "font-mono text-xs flex gap-4 items-start text-[11px] leading-7 font-medium",
                      "transition-all duration-300 ease-out",
                      isExpanded ? "translate-x-0 opacity-100" : "translate-x-2 opacity-0",
                      severityStyle,
                    )}
                    style={{
                      transitionDelay: isExpanded ? `${200 + index * 20}ms` : "0ms",
                    }}
                  >
                    <span>
                      <TimestampInfo
                        value={log.time}
                        className={cn("font-mono underline decoration-dotted p-0 text-[11px]")}
                      />
                    </span>
                    <span
                      className={cn(
                        "shrink-0 font-semibold uppercase tracking-wider",
                        severity === "ERROR"
                          ? "text-errorA-11"
                          : severity === "WARN"
                            ? "text-warningA-11"
                            : "text-grayA-11",
                      )}
                    >
                      [{SEVERITY_ABBR[severity] ?? "LOG"}]
                    </span>
                    <span className="shrink-0 font-medium  text-grayA-9">{log.region}</span>
                    <span className="max-w-[300px] min-w-0 truncate">{log.message}</span>
                    <span className="max-w-[300px] truncate">{JSON.stringify(log.attributes)}</span>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
