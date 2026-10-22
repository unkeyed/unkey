"use client";

import {
  type Grid,
  RANGE_PLANS,
  type RangePreset,
  latestGrid,
  nextTickMs,
} from "@/lib/metrics/app-metrics";
import { useEffect, useMemo, useState } from "react";

export function useBucketClock(range: RangePreset): Grid {
  const [clock, setClock] = useState(() => ({ range, nowMs: Date.now() }));
  if (clock.range !== range) {
    setClock({ range, nowMs: Date.now() });
  }

  const grid = useMemo(() => latestGrid(range, clock.nowMs), [range, clock.nowMs]);
  const live = RANGE_PLANS[range].live;
  const tickAtMs = nextTickMs(grid);

  useEffect(() => {
    if (!live) {
      return;
    }
    let timer: ReturnType<typeof setTimeout> | undefined;
    const tick = () => setClock({ range, nowMs: Date.now() });
    const schedule = () => {
      clearTimeout(timer);
      if (document.hidden) {
        return;
      }
      if (Date.now() >= tickAtMs) {
        tick();
        return;
      }
      timer = setTimeout(tick, tickAtMs - Date.now());
    };
    schedule();
    document.addEventListener("visibilitychange", schedule);
    return () => {
      clearTimeout(timer);
      document.removeEventListener("visibilitychange", schedule);
    };
  }, [live, range, tickAtMs]);

  return grid;
}
