"use client";

import { useSyncExternalStore } from "react";

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

// One clock for the whole page: labels reading their own Date.now() froze at
// mount, and a tooltip opening over a row disagreed with the row. The server
// snapshot is the moment the module loaded, so render this on the client only.
export function useNow(): number {
  return useSyncExternalStore(
    subscribe,
    () => now,
    () => now,
  );
}
