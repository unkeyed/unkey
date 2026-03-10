/**
 * Formats latency value with adaptive units
 * - <1000ms: shows "150ms"
 * - 1-59s: shows "5.0s"
 * - 1-59m: shows "1.5m"
 * - 1-24h: shows "2.0h"
 * - ≥1d: shows "1.2d"
 */
/**
 * Formats a duration in milliseconds as a compound string with up to 2 units.
 * - <1000ms → "500ms"
 * - 1s–59s → "45s"
 * - 1m–59m → "2m 20s" (drops seconds if 0 → "2m")
 * - 1h+ → "1h 15m" (drops minutes if 0 → "1h")
 */
export function formatCompoundDuration(ms: number): string {
  if (ms < 1000) {
    return `${Math.round(ms)}ms`;
  }

  const totalSeconds = Math.floor(ms / 1000);

  if (totalSeconds < 60) {
    return `${totalSeconds}s`;
  }

  const totalMinutes = Math.floor(totalSeconds / 60);

  if (totalMinutes < 60) {
    const remainingSeconds = totalSeconds % 60;
    return remainingSeconds > 0 ? `${totalMinutes}m ${remainingSeconds}s` : `${totalMinutes}m`;
  }

  const hours = Math.floor(totalMinutes / 60);
  const remainingMinutes = totalMinutes % 60;
  return remainingMinutes > 0 ? `${hours}h ${remainingMinutes}m` : `${hours}h`;
}

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
