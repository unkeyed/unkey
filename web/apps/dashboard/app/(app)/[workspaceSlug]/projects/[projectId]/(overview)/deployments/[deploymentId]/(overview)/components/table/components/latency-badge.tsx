import { cn } from "@/lib/utils";
import { formatLatency } from "@/lib/utils/metric-formatters";
import type { SentinelLogsResponse } from "@unkey/clickhouse/src/sentinel";
import { Badge, InfoTooltip } from "@unkey/ui";

export const LatencyBadge = ({ log }: { log: SentinelLogsResponse }) => {
  const style = getLatencyStyle(log.total_latency);
  const tooltipText = `Total: ${log.total_latency}ms | Instance: ${log.instance_latency}ms | Sentinel: ${log.sentinel_latency}ms`;

  return (
    <InfoTooltip content={tooltipText}>
      <Badge
        className={cn("px-[6px] rounded-md font-mono whitespace-nowrap tabular-nums", style)}
      >
        {formatLatency(log.total_latency)}
      </Badge>
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
    return "bg-error-4 text-error-11";
  }
  if (latency > 200) {
    return "bg-warning-4 text-warning-11";
  }
  return "bg-grayA-3 text-grayA-11 group-hover:bg-grayA-5";
};
