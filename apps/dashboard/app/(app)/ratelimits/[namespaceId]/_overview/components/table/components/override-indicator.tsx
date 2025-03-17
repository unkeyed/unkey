import { formatNumber } from "@/lib/fmt";
import { cn } from "@/lib/utils";
import type { RatelimitOverviewLog } from "@unkey/clickhouse/src/ratelimits";
import { ArrowDotAntiClockwise, Focus, TriangleWarning2 } from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";
import ms from "ms";
import { calculateBlockedPercentage } from "../utils/calculate-blocked-percentage";
import { getStatusStyle } from "../utils/get-row-class";
import { RatelimitOverviewTooltip } from "./ratelimit-overview-tooltip";

type IdentifierColumnProps = {
  log: RatelimitOverviewLog;
};

const PERCENTAGE_MULTIPLIER = 100;
const FULLY_BLOCKED_PERCENTAGE = 100;

export const IdentifierColumn = ({ log }: IdentifierColumnProps) => {
  const style = getStatusStyle(log);
  const hasMoreBlocked = calculateBlockedPercentage(log);
  const totalRequests = log.blocked_count + log.passed_count;
  const blockRate =
    totalRequests > 0 ? (log.blocked_count / totalRequests) * PERCENTAGE_MULTIPLIER : 0;
  const isFullyBlocked = blockRate === FULLY_BLOCKED_PERCENTAGE;

  return (
    <div className="flex gap-6 items-center pl-2">
      <RatelimitOverviewTooltip
        content={
          <p className="text-sm">
            {isFullyBlocked ? (
              "All requests have been blocked in this timeframe"
            ) : (
              <>
                More than {Math.round(blockRate)}% of requests have been
                <br />
                blocked in this timeframe
              </>
            )}
          </p>
        }
      >
        <div className={cn(hasMoreBlocked ? "block" : "invisible")}>
          <TriangleWarning2 />
        </div>
      </RatelimitOverviewTooltip>
      <div className="flex gap-3 items-center">
        <div
          className={cn(
            style.badge.default,
            "rounded p-1",
            hasMoreBlocked ? "" : "group-hover:bg-accent-6",
          )}
        >
          {log.override ? (
            <ArrowDotAntiClockwise size="md-regular" />
          ) : (
            <Focus
              size="md-regular"
              className={cn(hasMoreBlocked ? "" : "group-hover:text-accent-12")}
            />
          )}
        </div>
        <div
          className={cn("font-mono font-medium", hasMoreBlocked ? style.base : "text-accent-12")}
        >
          {log.identifier}
        </div>
        {log.override && <OverrideIndicator log={log} style={style} />}
      </div>
    </div>
  );
};

type OverrideIndicatorProps = {
  log: RatelimitOverviewLog;
  style: ReturnType<typeof getStatusStyle>;
};

const OverrideIndicator = ({ log, style }: OverrideIndicatorProps) => (
  <TooltipProvider>
    <Tooltip>
      <TooltipTrigger asChild>
        <div className="group relative p-3 cursor-pointer -ml-[5px]">
          <div className="absolute inset-0" />
          <div
            className={cn(
              "size-[6px] rounded-full absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2",
              calculateBlockedPercentage(log) ? "bg-orange-10 hover:bg-orange-11" : "bg-warning-10",
            )}
          />
        </div>
      </TooltipTrigger>
      <TooltipContent
        className="bg-gray-1 px-4 py-2 border border-gray-4 shadow-md font-medium text-xs text-accent-12 w-[265px]"
        side="right"
      >
        <div className="flex gap-3 items-center">
          <div
            className={cn(
              style.badge.default,
              "rounded p-1",
              "bg-accent-4 text-accent-11 group-hover:bg-accent-5",
            )}
          >
            <ArrowDotAntiClockwise size="md-regular" />
          </div>
          <div className="flex flex-col gap-1">
            <div className="text-sm flex gap-[10px] items-center">
              <span className="font-medium text-sm text-accent-12">Custom override in effect</span>
              <div className="size-[6px] rounded-full bg-warning-10" />
            </div>
            {log.override && (
              <div className="text-accent-9">
                Limit set to{" "}
                <span className="text-accent-12">{formatNumber(log.override.limit)} </span>
                requests per <span className="text-accent-12">{ms(log.override.duration)}</span>
              </div>
            )}
          </div>
        </div>
      </TooltipContent>
    </Tooltip>
  </TooltipProvider>
);
