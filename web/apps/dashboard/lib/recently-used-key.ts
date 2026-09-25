import { formatMs } from "@/lib/ms";

export const RECENTLY_USED_WINDOW_MS = 7 * 24 * 60 * 60 * 1000;

export const RECENTLY_USED_WINDOW_LABEL = "7 days";

export const isRecentlyUsed = (lastUsedAt: number, now: number = Date.now()): boolean =>
  lastUsedAt > 0 && now - lastUsedAt < RECENTLY_USED_WINDOW_MS;

export const formatTimeSinceLastUse = (lastUsedAt: number, now: number = Date.now()): string => {
  const elapsedMs = now - lastUsedAt;
  if (elapsedMs < 60_000) {
    return "less than a minute ago";
  }
  return `${formatMs(elapsedMs, { long: true })} ago`;
};
