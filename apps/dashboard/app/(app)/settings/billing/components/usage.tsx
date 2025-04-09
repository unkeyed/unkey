import { SettingCard } from "@/components/settings-card";
import { formatNumber } from "@/lib/fmt";

export const Usage: React.FC<{ current: number; max: number }> = async ({ current, max }) => {
  return (
    <SettingCard
      title="Usage this month"
      description="Valid key verifications and ratelimits."
      border="both"
    >
      <div className="w-full flex h-full items-center justify-end gap-4">
        <p className="text-sm font-semibold text-gray-12">
          {formatNumber(current)} / {formatNumber(max)} ({Math.round((current / max) * 100)}%)
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
  const offset = circumference - (safeValue / max) * circumference;
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
