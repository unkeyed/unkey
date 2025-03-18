"use client";
import { cn } from "@/lib/utils";
import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import { TriangleWarning2 } from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";
import { getErrorPercentage, getErrorSeverity } from "../utils/calculate-blocked-percentage";

export const KeyTooltip = ({
  children,
  content,
}: {
  children: React.ReactNode;
  content: React.ReactNode;
}) => {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>{children}</TooltipTrigger>
        <TooltipContent
          className="bg-gray-12 text-gray-1 px-3 py-2 border border-accent-6 shadow-md font-medium text-xs"
          side="right"
        >
          {content}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};

type KeyIdentifierColumnProps = {
  log: KeysOverviewLog;
};

// Get warning icon based on error severity
const getWarningIcon = (severity: string) => {
  switch (severity) {
    case "high":
      return <TriangleWarning2 className="text-error-11" />;
    case "moderate":
      return <TriangleWarning2 className="text-orange-11" />;
    case "low":
      return <TriangleWarning2 className="text-warning-11" />;
    default:
      return <TriangleWarning2 className="invisible" />;
  }
};

// Get tooltip message based on error severity
const getWarningMessage = (severity: string, errorRate: number) => {
  switch (severity) {
    case "high":
      return `Critical: ${Math.round(errorRate)}% of requests have failed`;
    case "moderate":
      return `Warning: ${Math.round(errorRate)}% of requests have failed`;
    case "low":
      return `${Math.round(errorRate)}% of requests have been invalid`;
    default:
      return "All requests are valid";
  }
};

export const KeyIdentifierColumn = ({ log }: KeyIdentifierColumnProps) => {
  const errorPercentage = getErrorPercentage(log);
  const severity = getErrorSeverity(log);
  const hasErrors = severity !== "none";

  return (
    <div className="flex gap-6 items-center pl-2">
      <KeyTooltip
        content={<p className="text-sm">{getWarningMessage(severity, errorPercentage)}</p>}
      >
        <div className={cn("transition-opacity", hasErrors ? "opacity-100" : "opacity-0")}>
          {getWarningIcon(severity)}
        </div>
      </KeyTooltip>
      <div className="font-mono font-medium truncate">
        {log.key_id.substring(0, 8)}...
        {log.key_id.substring(log.key_id.length - 4)}
      </div>
    </div>
  );
};
