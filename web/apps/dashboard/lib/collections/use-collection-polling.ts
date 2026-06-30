"use client";

import { useEffect, useRef } from "react";

/**
 * Calls `refetch` on an interval only while `enabled`.
 *
 * Collections no longer carry an always-on refetchInterval, so liveness is
 * opt-in by the component that actually displays changing data (in-flight
 * deploys, pending domain verification). Idle pages pass enabled:false and rely
 * on refetchOnWindowFocus + post-mutation refetch for freshness.
 *
 * Pass a stable callback (e.g. `() => collection.x.utils.refetch()`); it is read
 * through a ref so the interval is not torn down when the callback identity
 * changes between renders.
 */
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
    const id = setInterval(() => refetchRef.current(), intervalMs);
    return () => clearInterval(id);
  }, [intervalMs, enabled]);
}
