"use client";

import { useEffect, useRef } from "react";

const TAB_RETURN_MIN_GAP_MS = 5_000;

export function useCollectionPolling(
  refetch: () => void,
  { intervalMs, enabled }: { intervalMs: number; enabled: boolean },
): void {
  const refetchRef = useRef(refetch);
  refetchRef.current = refetch;

  useEffect(() => {
    if (!enabled) {
      return;
    }

    let lastRefetchAtMs = Date.now();
    const refetchNow = () => {
      lastRefetchAtMs = Date.now();
      refetchRef.current();
    };

    const id = setInterval(() => {
      if (!document.hidden) {
        refetchNow();
      }
    }, intervalMs);

    const onVisibilityChange = () => {
      if (
        !document.hidden &&
        Date.now() - lastRefetchAtMs >= Math.min(intervalMs, TAB_RETURN_MIN_GAP_MS)
      ) {
        refetchNow();
      }
    };
    document.addEventListener("visibilitychange", onVisibilityChange);

    return () => {
      clearInterval(id);
      document.removeEventListener("visibilitychange", onVisibilityChange);
    };
  }, [intervalMs, enabled]);
}
