import { formatNumber } from "@/lib/fmt";

export const MetricStats = ({
  successCount,
  errorCount,
  successLabel = "VALID",
  errorLabel = "INVALID",
}: {
  successCount: number;
  errorCount: number;
  successLabel?: string;
  errorLabel?: string;
}) => (
  <div className="flex gap-[14px] items-center">
    <div className="flex flex-col gap-1">
      <div className="flex gap-2 items-center">
        <div className="bg-accent-8 rounded h-[10px] w-1" />
        <div className="text-accent-12 text-xs font-medium">{formatNumber(successCount)}</div>
        <div className="text-accent-9 text-[11px] leading-4">{successLabel}</div>
      </div>
    </div>
    <div className="flex flex-col gap-1">
      <div className="flex gap-2 items-center">
        <div className="bg-orange-9 rounded h-[10px] w-1" />
        <div className="text-accent-12 text-xs font-medium">{formatNumber(errorCount)}</div>
        <div className="text-accent-9 text-[11px] leading-4">{errorLabel}</div>
      </div>
    </div>
  </div>
);
