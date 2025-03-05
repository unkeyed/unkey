"use client";

import { cn } from "@/lib/utils";
import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import { TriangleWarning2 } from "@unkey/icons";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@unkey/ui";
import { calculateErrorPercentage } from "../utils/calculate-blocked-percentage";

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

export const KeyIdentifierColumn = ({ log }: KeyIdentifierColumnProps) => {
  const hasHighErrorRate = calculateErrorPercentage(log);
  const totalRequests = log.valid_count + log.error_count;
  const errorRate =
    totalRequests > 0 ? (log.error_count / totalRequests) * 100 : 0;

  return (
    <div className="flex gap-6 items-center pl-2">
      <KeyTooltip
        content={
          <p className="text-sm">
            More than {Math.round(errorRate)}% of requests have been
            <br />
            invalid in this timeframe
          </p>
        }
      >
        <div className={cn(hasHighErrorRate ? "block" : "invisible")}>
          <TriangleWarning2 />
        </div>
      </KeyTooltip>
      <div className="flex gap-3 items-center">
        <div className="font-mono text-accent-12 font-medium truncate">
          {log.key_id.substring(0, 8)}...
          {log.key_id.substring(log.key_id.length - 4)}
        </div>
      </div>
    </div>
  );
};
