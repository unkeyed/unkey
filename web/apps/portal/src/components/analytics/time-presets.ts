export const MINUTE_MS = 60_000;
export const HOUR_MS = 3_600_000;
export const DAY_MS = 86_400_000;

export const TIME_PRESETS = [
  { id: "1h", label: "Last 1 hour", ms: HOUR_MS },
  { id: "6h", label: "Last 6 hours", ms: 6 * HOUR_MS },
  { id: "12h", label: "Last 12 hours", ms: 12 * HOUR_MS },
  { id: "24h", label: "Last 24 hours", ms: DAY_MS },
  { id: "2d", label: "Last 2 days", ms: 2 * DAY_MS },
  { id: "7d", label: "Last 7 days", ms: 7 * DAY_MS },
  { id: "30d", label: "Last 30 days", ms: 30 * DAY_MS },
] as const;

export type TimePreset = (typeof TIME_PRESETS)[number];
export type TimePresetId = TimePreset["id"];
export const TIME_PRESET_IDS = TIME_PRESETS.map((p) => p.id) as [TimePresetId, ...TimePresetId[]];

const PREFERRED_PRESET: TimePresetId = "7d";

/**
 * The presets a workspace can query. `portal.getVerifications` rejects any
 * window wider than the workspace's log retention, so longer presets are
 * hidden. Non-positive retention means the API skips the check.
 */
export function availableTimePresets(retentionDays: number): TimePreset[] {
  if (retentionDays <= 0) {
    return [...TIME_PRESETS];
  }
  return TIME_PRESETS.filter((p) => p.ms <= retentionDays * DAY_MS);
}

/** The 7-day view when retention allows it, otherwise the longest preset that fits. */
export function defaultTimePreset(retentionDays: number): TimePreset {
  const available = availableTimePresets(retentionDays);
  // Retention under an hour leaves nothing to offer, so fall back to the
  // shortest window rather than hand the caller an undefined it can't render.
  return (
    available.find((p) => p.id === PREFERRED_PRESET) ??
    available[available.length - 1] ??
    TIME_PRESETS[0]
  );
}

export type TimeWindow = { startTime: number; endTime: number; days: number };

/**
 * Anchors the window to the minute so every consumer that resolves the same
 * preset in the same minute lands on the same query key.
 */
export function resolveWindow(preset: TimePreset, now: number): TimeWindow {
  const endTime = Math.floor(now / MINUTE_MS) * MINUTE_MS;
  return { startTime: endTime - preset.ms, endTime, days: preset.ms / DAY_MS };
}
