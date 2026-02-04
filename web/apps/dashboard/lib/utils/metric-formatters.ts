/**
 * Formats latency value with adaptive units
 * - <1000ms: shows "150ms"
 * - 1-59s: shows "5.0s"
 * - 1-59m: shows "1.5m"
 * - 1-24h: shows "2.0h"
 * - ≥1d: shows "1.2d"
 */
export function formatLatency(latency: number): string {
  if (latency < 1000) {
    return `${latency}ms`;
  }

  const seconds = latency / 1000;
  if (seconds < 60) {
    return `${seconds.toFixed(1)}s`;
  }

  const minutes = seconds / 60;
  if (minutes < 60) {
    return `${minutes.toFixed(1)}m`;
  }

  const hours = minutes / 60;
  if (hours < 24) {
    return `${hours.toFixed(1)}h`;
  }

  const days = hours / 24;
  return `${days.toFixed(1)}d`;
}
