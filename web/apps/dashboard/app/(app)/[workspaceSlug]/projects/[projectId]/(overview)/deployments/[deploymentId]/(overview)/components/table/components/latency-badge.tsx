import { cn } from "@/lib/utils";
import { formatLatency } from "@/lib/utils/metric-formatters";
import type { SentinelLogsResponse } from "@unkey/clickhouse/src/sentinel";
import { InfoTooltip } from "@unkey/ui";

export const LatencyBadge = ({ log }: { log: SentinelLogsResponse }) => {
  const style = getLatencyStyle(log.total_latency);
  const tooltipText = `Total: ${log.total_latency}ms | Instance: ${log.instance_latency}ms | Sentinel: ${log.sentinel_latency}ms`;

  return (
    <InfoTooltip content={tooltipText}>
      <span className={cn("px-[6px] font-mono whitespace-nowrap tabular-nums", style)}>
        {formatLatency(log.total_latency)}
      </span>
    </InfoTooltip>
  );
};

/**
 * Get CSS classes for latency badge based on performance thresholds
 * - >500ms: error (red)
 * - >200ms: warning (yellow)
 * - Otherwise: default (gray)
 */
const getLatencyStyle = (latency: number): string => {
  if (latency > 500) {
    return "text-error-11";
  }
  if (latency > 200) {
    return "text-warning-11";
  }
  return "text-grayA-11";
};
