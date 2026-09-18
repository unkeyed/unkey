"use client";

import { useCallback, useSyncExternalStore } from "react";
import { type RelativeStyle, elapsed, relativeTime } from "../lib/time";

const TICK_MS = 1_000;

const listeners = new Set<() => void>();
let now = Date.now();
let timer: ReturnType<typeof setInterval> | undefined;

function tick(): void {
  now = Date.now();
  for (const notify of listeners) {
    notify();
  }
}

function subscribe(listener: () => void): () => void {
  if (listeners.size === 0) {
    now = Date.now();
    timer = setInterval(tick, TICK_MS);
  }
  listeners.add(listener);

  return () => {
    listeners.delete(listener);
    if (listeners.size === 0) {
      clearInterval(timer);
      timer = undefined;
    }
  };
}

// One clock for every label on the page, so two of them cannot disagree, and a
// label ages on its own instead of waiting for an unrelated render. A label
// whose text has not changed re-reads the clock but does not re-render, which
// is what keeps a table of hundreds of rows cheap. The server snapshot is the
// moment the module loaded, so render these on the client only.
function useClockedLabel(read: () => string): string {
  return useSyncExternalStore(subscribe, read, read);
}

export function useRelativeTime(
  value: string | number | Date,
  style: RelativeStyle = "long",
): string {
  const time = value instanceof Date ? value.getTime() : value;
  return useClockedLabel(useCallback(() => relativeTime(time, now, style), [time, style]));
}

export function useElapsed(value: string | number | Date, style: RelativeStyle = "long"): string {
  const time = value instanceof Date ? value.getTime() : value;
  return useClockedLabel(useCallback(() => elapsed(time, now, style), [time, style]));
}
