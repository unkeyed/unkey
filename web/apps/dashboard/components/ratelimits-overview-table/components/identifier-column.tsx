import { formatNumber } from "@/lib/fmt";
import { formatMs } from "@/lib/ms";
import type { RatelimitOverviewLog } from "@unkey/clickhouse/src/ratelimits";
import {
  IconArrowDotRotateAnticlockwiseOutline18,
  IconFocusOutline18,
  IconTriangleWarningOutline18,
} from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
import { cn } from "cn";
import { getBlockedPercentage, isMostlyBlocked } from "../utils/calculate-blocked-percentage";
import { getStatusStyle } from "../utils/get-row-class";

type IdentifierColumnProps = {
  log: RatelimitOverviewLog;
};

export const IdentifierColumn = ({ log }: IdentifierColumnProps) => {
  const style = getStatusStyle(log);
  const hasMoreBlocked = isMostlyBlocked(log);
  const blockedPercent = getBlockedPercentage(log);
  const isFullyBlocked = blockedPercent === 100;

  return (
    <div className="flex gap-6 items-center pl-2 min-w-0">
      <InfoTooltip
        content={
          <div className="text-xs">
            {isFullyBlocked ? (
              "All requests have been blocked in this timeframe"
            ) : (
              <>
                More than {Math.round(blockedPercent)}% of requests have been
                <br />
                blocked in this timeframe
              </>
            )}
          </div>
        }
      >
        <div className={cn(hasMoreBlocked ? "flex items-center shrink-0" : "invisible shrink-0")}>
          <IconTriangleWarningOutline18 className="size-3.5" />
        </div>
      </InfoTooltip>
      <div className="flex gap-3 items-center min-w-0">
        <div
          className={cn(
            style.badge.default,
            "rounded-sm p-1 shrink-0",
            hasMoreBlocked ? "" : "group-hover:bg-gray-6",
          )}
        >
          {log.override ? (
            <IconArrowDotRotateAnticlockwiseOutline18 className="size-3.5" />
          ) : (
            <IconFocusOutline18
              className={cn("size-3.5", hasMoreBlocked ? "" : "group-hover:text-gray-12")}
            />
          )}
        </div>
        <InfoTooltip
          asChild
          content={<span className="font-mono text-xs break-all">{log.identifier}</span>}
        >
          <div
            className={cn(
              "font-mono font-medium truncate min-w-0",
              hasMoreBlocked ? style.base : "text-gray-12",
            )}
          >
            {log.identifier}
          </div>
        </InfoTooltip>
        {log.override && (
          <OverrideIndicator log={log} style={style} hasMoreBlocked={hasMoreBlocked} />
        )}
      </div>
    </div>
  );
};

type OverrideIndicatorProps = {
  log: RatelimitOverviewLog;
  style: ReturnType<typeof getStatusStyle>;
  hasMoreBlocked: boolean;
};

const OverrideIndicator = ({ log, style, hasMoreBlocked }: OverrideIndicatorProps) => (
  <InfoTooltip
    content={
      <div className="flex flex-row pl-1 pr-5 gap-3 py-0 items-center justify-center leading-none">
        <div className={cn(style.badge.default, "rounded-sm p-1", "bg-gray-1/15")}>
          <IconArrowDotRotateAnticlockwiseOutline18 className="size-3.5" />
        </div>
        <div className="flex flex-col gap-1">
          <div className="text-sm flex gap-[10px] items-center">
            <span className="font-medium text-sm">Custom override in effect</span>
            <div className="size-[6px] rounded-full bg-warning-10" />
          </div>
          {log.override && (
            <div className="text-xs">
              <span className="opacity-75">Limit set to</span> {formatNumber(log.override.limit)}{" "}
              <span className="opacity-75">requests per</span> {formatMs(log.override.duration)}
            </div>
          )}
        </div>
      </div>
    }
    asChild
  >
    <div className="group relative p-3 cursor-pointer -ml-[5px]">
      <div className="absolute inset-0" />
      <div
        className={cn(
          "size-[6px] rounded-full absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2",
          hasMoreBlocked ? "bg-orange-10 hover:bg-orange-11" : "bg-warning-10",
        )}
      />
    </div>
  </InfoTooltip>
);
