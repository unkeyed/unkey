"use client";

import { useEffect, useRef } from "react";

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

    const id = setInterval(() => {
      if (!document.hidden) {
        refetchRef.current();
      }
    }, intervalMs);

    const onVisibilityChange = () => {
      if (!document.hidden) {
        refetchRef.current();
      }
    };
    document.addEventListener("visibilitychange", onVisibilityChange);

    return () => {
      clearInterval(id);
      document.removeEventListener("visibilitychange", onVisibilityChange);
    };
  }, [intervalMs, enabled]);
}
