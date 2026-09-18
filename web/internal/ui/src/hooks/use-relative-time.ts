"use client";

import { useCallback, useSyncExternalStore } from "react";
import { type RelativeStyle, parseTimestamp, relativeTime } from "../lib/time";

export type { RelativeStyle };

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

// Background tabs throttle setInterval, so force a tick when the tab becomes visible again.
function tickIfVisible(): void {
  if (document.visibilityState === "visible") {
    tick();
  }
}

function subscribe(listener: () => void): () => void {
  if (listeners.size === 0) {
    now = Date.now();
    timer = setInterval(tick, TICK_MS);
    document.addEventListener("visibilitychange", tickIfVisible);
  }
  listeners.add(listener);

  return () => {
    listeners.delete(listener);
    if (listeners.size === 0) {
      clearInterval(timer);
      timer = undefined;
      document.removeEventListener("visibilitychange", tickIfVisible);
    }
  };
}

function clock(): number {
  return timer === undefined ? Date.now() : now;
}

export function useRelativeTime(
  value: string | number | Date,
  style: RelativeStyle = "narrow",
): string {
  const time = parseTimestamp(value).getTime();
  const read = useCallback(() => relativeTime(time, clock(), style), [time, style]);
  return useSyncExternalStore(subscribe, read, read);
}

// A server timestamp read against the browser's clock can sit ahead of it, and
// something that has already happened must never read "in 25 sec".
export function useElapsed(value: string | number | Date, style: RelativeStyle = "narrow"): string {
  const time = parseTimestamp(value).getTime();
  const read = useCallback(() => relativeTime(time, Math.max(clock(), time), style), [time, style]);
  return useSyncExternalStore(subscribe, read, read);
}
