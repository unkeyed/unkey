"use client";

import { useEffect, useRef } from "react";

const TAB_RETURN_MIN_GAP_MS = 5_000;

export function useCollectionPolling(
  refetch: () => void,
  { intervalMs, enabled }: { intervalMs: number; enabled: boolean },
): void {
  const refetchRef = useRef(refetch);
  refetchRef.current = refetch;
  // Survives interval changes, so a new cadence counts from the last refetch
  // instead of restarting the wait
  const lastRefetchAtMs = useRef<number | null>(null);

  useEffect(() => {
    if (!enabled) {
      lastRefetchAtMs.current = null;
      return;
    }
    lastRefetchAtMs.current ??= Date.now();

    let timer: ReturnType<typeof setTimeout> | undefined;
    const refetchNow = () => {
      lastRefetchAtMs.current = Date.now();
      refetchRef.current();
    };
    const schedule = () => {
      const lastAtMs = lastRefetchAtMs.current ?? Date.now();
      timer = setTimeout(
        () => {
          // A hidden tab stops here; becoming visible schedules again
          if (document.hidden) {
            return;
          }
          refetchNow();
          schedule();
        },
        Math.max(0, lastAtMs + intervalMs - Date.now()),
      );
    };

    const onVisibilityChange = () => {
      if (document.hidden) {
        return;
      }
      clearTimeout(timer);
      if (
        Date.now() - (lastRefetchAtMs.current ?? 0) >=
        Math.min(intervalMs, TAB_RETURN_MIN_GAP_MS)
      ) {
        refetchNow();
      }
      schedule();
    };

    schedule();
    document.addEventListener("visibilitychange", onVisibilityChange);

    return () => {
      clearTimeout(timer);
      document.removeEventListener("visibilitychange", onVisibilityChange);
    };
  }, [intervalMs, enabled]);
}
