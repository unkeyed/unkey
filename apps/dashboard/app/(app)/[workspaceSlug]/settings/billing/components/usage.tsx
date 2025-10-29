"use client";
import { formatNumber } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import { SettingCard } from "@unkey/ui";

export const Usage: React.FC<{
  quota: number;
}> = ({ quota }) => {
  const { data: usage, isLoading, error, refetch } = trpc.billing.queryUsage.useQuery(undefined, {
    // Ensure fresh data when workspace changes
    staleTime: 0,
  });

  if (isLoading) {
    return (
      <SettingCard
        title="Usage this month"
        description="Valid key verifications and ratelimits."
        border="both"
        className="w-full"
        contentWidth="w-full lg:w-[320px]"
      >
        <div className="w-full flex h-full items-center justify-end gap-4">
          <div className="h-5 w-32 bg-gray-4 animate-pulse rounded" />
          <div className="h-6 w-6 bg-gray-4 animate-pulse rounded-full" />
        </div>
      </SettingCard>
    );
  }

  if (error) {
    return (
      <SettingCard
        title="Usage this month"
        description="Valid key verifications and ratelimits."
        border="both"
        className="w-full"
        contentWidth="w-full lg:w-[320px]"
      >
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
      </SettingCard>
    );
  }

  if (!usage) {
    return null;
  }

  const verifications = usage.billableVerifications;
  const ratelimits = usage.billableRatelimits;
  const current = verifications + ratelimits;
  const max = quota;
  const percent = max > 0 ? Math.round((current / max) * 100) : 0;

  return (
    <SettingCard
      title="Usage this month"
      description="Valid key verifications and ratelimits."
      border="both"
      className="w-full"
      contentWidth="w-full lg:w-[320px]"
    >
      <div className="w-full flex h-full items-center justify-end gap-4">
        <p className="text-sm font-semibold text-gray-12">
          {formatNumber(current)} / {formatNumber(max)} ({percent}%)
        </p>

        <ProgressCircle max={max} value={current} />
      </div>
    </SettingCard>
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
