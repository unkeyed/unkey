import { useMemo } from "react";
import { type TimePreset, resolveWindow } from "~/components/analytics/time-presets";

/** Resolves the preset against the clock once per selection, not per render. */
export function useTimeWindow(preset: TimePreset) {
  return useMemo(() => resolveWindow(preset, Date.now()), [preset]);
}
