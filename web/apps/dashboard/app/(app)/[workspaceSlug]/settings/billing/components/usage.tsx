"use client";
import { formatNumber } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import { BillingCard } from "./billing-card";

export const Usage: React.FC<{
  quota: number;
}> = ({ quota }) => {
  const {
    data: usage,
    isLoading,
    error,
    refetch,
  } = trpc.billing.queryUsage.useQuery(undefined, {
    // Cache for 30 seconds to reduce unnecessary refetches
    // TRPC automatically scopes by workspace via requireWorkspace middleware
    staleTime: 30_000, // 30 seconds
    // Skip batching to prevent analytics slowdown from blocking core UI
    trpc: {
      context: {
        skipBatch: true,
      },
    },
    retry: 1,
  });

  if (isLoading) {
    return (
      <BillingCard label="Usage this month" description="Valid key verifications and ratelimits.">
        <div className="w-full flex h-full items-center justify-end gap-4">
          <div className="h-5 w-32 bg-gray-4 animate-pulse rounded-sm" />
          <div className="h-6 w-6 bg-gray-4 animate-pulse rounded-full" />
        </div>
      </BillingCard>
    );
  }

  if (error) {
    return (
      <BillingCard label="Usage this month" description="Valid key verifications and ratelimits.">
        <div className="w-full flex flex-col gap-2">
          <p className="text-sm text-red-11">Failed to load usage: {error.message}</p>
          <button
            type="button"
            onClick={() => refetch()}
            className="text-sm text-accent-11 hover:text-accent-12 transition-colors text-left"
          >
            Retry
          </button>
        </div>
      </BillingCard>
    );
  }

  if (!usage) {
    return (
      <BillingCard label="Usage this month" description="Valid key verifications and ratelimits.">
        <div className="w-full flex flex-col gap-2">
          <p className="text-sm text-gray-11">No usage data available</p>
        </div>
      </BillingCard>
    );
  }

  // Safely extract and validate numeric values with fallbacks
  const verifications =
    typeof usage.billableVerifications === "number" && !Number.isNaN(usage.billableVerifications)
      ? usage.billableVerifications
      : 0;
  const ratelimits =
    typeof usage.billableRatelimits === "number" && !Number.isNaN(usage.billableRatelimits)
      ? usage.billableRatelimits
      : 0;
  const current = verifications + ratelimits;
  const max = quota;
  const percent = max > 0 ? Math.round((current / max) * 100) : 0;
  const barPercent = clamp(0, percent, 100);

  return (
    <BillingCard
      label="Usage this month"
      description="Valid key verifications and ratelimits."
      footer={
        // Decorative: the exact numbers are rendered as text in the row above.
        <div className="h-1 w-full bg-grayA-3" aria-hidden>
          <div
            className={percent >= 100 ? "h-full bg-error-9" : "h-full bg-gray-12"}
            style={{ width: `${barPercent}%` }}
          />
        </div>
      }
    >
      <p className="font-mono text-[13px] text-gray-12">
        {formatNumber(current)} / {formatNumber(max)}{" "}
        <span className="text-gray-9">({percent}%)</span>
      </p>
    </BillingCard>
  );
};
function clamp(min: number, value: number, max: number): number {
  return Math.min(max, Math.max(value, min));
}

export const ProgressCircle: React.FC<{
  value: number;
  max: number;
  color?: string;
}> = ({ value, max, color }) => {
  const safeValue = clamp(0, value, max);
  const radius = 12;
  const strokeWidth = 3;
  const normalizedRadius = radius - strokeWidth / 2;
  const circumference = normalizedRadius * 2 * Math.PI;
  const offset = max > 0 ? circumference - (safeValue / max) * circumference : circumference;
  return (
    <>
      <div className="relative flex items-center justify-center">
        <svg
          width={radius * 2}
          height={radius * 2}
          viewBox={`0 0 ${radius * 2} ${radius * 2}`}
          className="-rotate-90 transform"
          aria-label="progress bar"
          aria-valuenow={value}
          aria-valuemax={max}
          data-max={max}
          data-value={safeValue ?? null}
        >
          <circle
            r={normalizedRadius}
            cx={radius}
            cy={radius}
            strokeWidth={strokeWidth}
            fill="transparent"
            stroke=""
            strokeLinecap="round"
            className="transition-colors ease-linear stroke-gray-4"
          />
          {safeValue >= 0 ? (
            <circle
              r={normalizedRadius}
              cx={radius}
              cy={radius}
              strokeWidth={strokeWidth}
              strokeDasharray={`${circumference} ${circumference}`}
              strokeDashoffset={offset}
              fill="transparent"
              stroke=""
              strokeLinecap="round"
              className="stroke-accent-12 transform-gpu transition-all duration-300 ease-in-out"
              style={{ stroke: color }}
            />
          ) : null}
        </svg>
      </div>
    </>
  );
};
