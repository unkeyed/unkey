import { formatNumber } from "@/lib/fmt";

import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";
import { useMemo } from "react";
import type { ProcessedTimeseriesDataPoint } from "../use-fetch-timeseries";

type OutcomeExplainerProps = {
  children: React.ReactNode;
  timeseries: ProcessedTimeseriesDataPoint[];
};

type ErrorType = {
  type: string;
  value: number;
  color: string;
};

export function OutcomeExplainer({ children, timeseries }: OutcomeExplainerProps): JSX.Element {
  // Aggregate all timeseries data for the tooltip
  const aggregatedData = useMemo(() => {
    if (!timeseries || timeseries.length === 0) {
      return {
        valid: 0,
        rate_limited: 0,
        insufficient_permissions: 0,
        forbidden: 0,
        disabled: 0,
        expired: 0,
        usage_exceeded: 0,
        total: 0,
      };
    }

    return timeseries.reduce(
      (acc, dataPoint) => {
        acc.valid += dataPoint.valid || 0;
        acc.rate_limited += dataPoint.rate_limited || 0;
        acc.insufficient_permissions += dataPoint.insufficient_permissions || 0;
        acc.forbidden += dataPoint.forbidden || 0;
        acc.disabled += dataPoint.disabled || 0;
        acc.expired += dataPoint.expired || 0;
        acc.usage_exceeded += dataPoint.usage_exceeded || 0;
        acc.total += dataPoint.total || 0;
        return acc;
      },
      {
        valid: 0,
        rate_limited: 0,
        insufficient_permissions: 0,
        forbidden: 0,
        disabled: 0,
        expired: 0,
        usage_exceeded: 0,
        total: 0,
      },
    );
  }, [timeseries]);

  const errorTypes = useMemo(() => {
    return [
      {
        type: "Insufficient Permissions",
        value: formatNumber(aggregatedData.insufficient_permissions) || 0,
        color: "bg-error-9",
      },
      {
        type: "Rate Limited",
        value: formatNumber(aggregatedData.rate_limited) || 0,
        color: "bg-error-9",
      },
      {
        type: "Forbidden",
        value: formatNumber(aggregatedData.forbidden) || 0,
        color: "bg-error-9",
      },
      {
        type: "Disabled",
        value: formatNumber(aggregatedData.disabled) || 0,
        color: "bg-error-9",
      },
      {
        type: "Expired",
        value: formatNumber(aggregatedData.expired) || 0,
        color: "bg-error-9",
      },
      {
        type: "Usage Exceeded",
        value: formatNumber(aggregatedData.usage_exceeded) || 0,
        color: "bg-error-9",
      },
    ].filter((error) => Number(error.value) > 0) as ErrorType[];
  }, [aggregatedData]);

  return (
    <TooltipProvider>
      <Tooltip delayDuration={300}>
        <TooltipTrigger asChild>
          <div>{children}</div>
        </TooltipTrigger>
        <TooltipContent
          className="min-w-64 bg-gray-1 dark:bg-black shadow-2xl p-0 border border-grayA-2 rounded-lg overflow-hidden flex justify-start px-4 pt-2 pb-1 flex-col gap-1"
          side="bottom"
        >
          <div className="text-gray-12 font-medium text-[13px] pr-2">API Key Activity</div>
          <div className="text-xs text-grayA-9 pr-2 font-normal">Last 36 hours</div>

          {/* Valid count */}
          <div className="flex justify-between w-full items-center mt-3">
            <div className="flex gap-3 items-center">
              <div className="bg-gray-7 h-6 w-0.5 rounded-t rounded-b" />
              <div className="text-gray-12 font-medium text-[13px]">Valid</div>
            </div>
            <div className="text-gray-9 font-medium text-[13px]">
              {formatNumber(aggregatedData.valid)}
            </div>
          </div>

          <div className="mt-1" />

          {/* Error types */}
          <div className="flex flex-col">
            {errorTypes.map((error, index) => (
              <div
                // biome-ignore lint/suspicious/noArrayIndexKey: <explanation>
                key={index}
                className="flex justify-between w-full items-center"
              >
                <div className="flex gap-3 items-center">
                  <div className={`${error.color} h-6 w-0.5 rounded-t rounded-b`} />
                  <div className="text-gray-12 font-medium text-[13px]">{error.type}</div>
                </div>
                <div className="text-gray-9 font-medium text-[13px]">{error.value}</div>
              </div>
            ))}

            {errorTypes.length === 0 && aggregatedData.valid === 0 && (
              <div className="text-gray-9 text-[13px] py-1">No verification activity</div>
            )}
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
