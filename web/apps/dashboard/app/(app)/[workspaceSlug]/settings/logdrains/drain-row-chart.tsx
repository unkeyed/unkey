"use client";

import { formatNumber } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import { InfoTooltip, Skeleton } from "@unkey/ui";
import { useMemo } from "react";

const HOURS = 168;
const MAX_BARS = 30;
const MAX_BAR_HEIGHT = 28;
const MAX_HEIGHT_BUFFER_FACTOR = 1.3;

type DeliveryBucket = {
  ts: number;
  successCount: number;
  transientErrorCount: number;
  permanentErrorCount: number;
};

type Bar = {
  id: number;
  successHeight: number;
  failureHeight: number;
  successCount: number;
  failureCount: number;
};

export function DrainRowChart({ drainId }: { drainId: string }) {
  const metrics = trpc.logdrain.metrics.useQuery({ drainId, hours: HOURS });
  const bars = useDeliveryBars(metrics.data?.series);
  const successCount = bars.reduce((total, bar) => total + bar.successCount, 0);
  const failureCount = bars.reduce((total, bar) => total + bar.failureCount, 0);

  if (metrics.isLoading) {
    return <Skeleton className="h-7 w-[158px] rounded-md" />;
  }

  if (metrics.isError) {
    return <ChartMessage className="text-error-9">Unable to load</ChartMessage>;
  }

  if (successCount + failureCount === 0) {
    return <ChartMessage className="text-grayA-9">No delivery attempts</ChartMessage>;
  }

  return (
    <InfoTooltip
      asChild
      delayDuration={300}
      variant="inverted"
      position={{ side: "bottom" }}
      className="rounded-lg border border-grayA-2 bg-gray-1 p-0 shadow-2xl dark:bg-black"
      content={
        <div className="flex min-w-56 flex-col gap-3 px-4 py-3">
          <div>
            <div className="text-[13px] font-medium text-gray-12">Delivery activity</div>
            <div className="text-xs font-normal text-grayA-9">Past 7 days</div>
          </div>
          <div className="flex flex-col gap-2">
            <ChartTotal
              colorClassName="bg-gray-7"
              label="Successful attempts"
              value={successCount}
            />
            <ChartTotal colorClassName="bg-error-9" label="Failed attempts" value={failureCount} />
          </div>
        </div>
      }
    >
      <div
        className="grid h-7 w-[158px] items-end gap-0.5 overflow-hidden rounded-md border border-transparent bg-grayA-2 px-1"
        style={{ gridTemplateColumns: `repeat(${MAX_BARS}, 3px)` }}
        aria-label={`${formatNumber(successCount)} successful and ${formatNumber(failureCount)} failed delivery attempts in the past 7 days`}
      >
        {bars.map((bar) => (
          <div key={bar.id} className="flex flex-col" aria-hidden="true">
            <div className="w-[3px] bg-error-9" style={{ height: bar.failureHeight }} />
            <div className="w-[3px] bg-grayA-5" style={{ height: bar.successHeight }} />
          </div>
        ))}
      </div>
    </InfoTooltip>
  );
}

function useDeliveryBars(series?: DeliveryBucket[]): Bar[] {
  return useMemo(() => {
    if (!series) {
      return [];
    }

    const end = Date.now();
    const start = end - HOURS * 60 * 60_000;
    const bucketMs = (end - start) / MAX_BARS;
    const attempts = Array.from({ length: MAX_BARS }, (_, index) => ({
      id: start + index * bucketMs,
      successCount: 0,
      failureCount: 0,
    }));

    for (const item of series) {
      const index = Math.max(
        0,
        Math.min(MAX_BARS - 1, Math.floor((Number(item.ts) - start) / bucketMs)),
      );
      attempts[index].successCount += Number(item.successCount);
      attempts[index].failureCount +=
        Number(item.transientErrorCount) + Number(item.permanentErrorCount);
    }

    const maxAttempts =
      Math.max(...attempts.map((bar) => bar.successCount + bar.failureCount), 1) *
      MAX_HEIGHT_BUFFER_FACTOR;
    return attempts.map((bar): Bar => {
      const totalCount = bar.successCount + bar.failureCount;
      const totalHeight = Math.min(
        Math.round((totalCount / maxAttempts) * MAX_BAR_HEIGHT),
        MAX_BAR_HEIGHT,
      );
      const failureHeight = bar.failureCount
        ? Math.max(Math.round((bar.failureCount / totalCount) * totalHeight), 1)
        : 0;
      return {
        ...bar,
        failureHeight,
        successHeight: Math.max(totalHeight - failureHeight, 0),
      };
    });
  }, [series]);
}

function ChartMessage({ children, className }: { children: React.ReactNode; className: string }) {
  return (
    <div
      className={`flex h-7 w-[158px] items-center justify-center rounded-md border border-transparent bg-grayA-2 px-2 text-xs ${className}`}
    >
      {children}
    </div>
  );
}

function ChartTotal({
  colorClassName,
  label,
  value,
}: {
  colorClassName: string;
  label: string;
  value: number;
}) {
  return (
    <div className="flex items-center justify-between gap-6">
      <div className="flex items-center gap-3">
        <div className={`h-5 w-0.5 rounded-full ${colorClassName}`} />
        <span className="text-[13px] font-medium text-gray-12">{label}</span>
      </div>
      <span className="text-[13px] font-medium text-gray-9">{formatNumber(value)}</span>
    </div>
  );
}
