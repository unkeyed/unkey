import type { RatelimitOverviewLog } from "@unkey/clickhouse/src/ratelimits";
import { TriangleWarning2 } from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { calculateBlockedPercentage } from "../utils/calculate-blocked-percentage";

export const WarningWithTooltip = ({ log }: { log: RatelimitOverviewLog }) => {
  const totalRequests = log.blocked_count + log.passed_count;
  const blockRate = totalRequests > 0 ? (log.blocked_count / totalRequests) * 100 : 0;

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <div className={cn(calculateBlockedPercentage(log) ? "block" : "invisible")}>
            <TriangleWarning2 />
          </div>
        </TooltipTrigger>
        <TooltipContent
          className="bg-gray-12 text-gray-1 px-3 py-2 border border-accent-6 shadow-md font-medium text-xs"
          side="right"
        >
          <p className="text-sm">
            More than {Math.round(blockRate)}% of requests have been
            <br />
            blocked in this timeframe
          </p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};
