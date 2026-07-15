"use client";

import { useCallback, useEffect, useRef, useState } from "react";

export type PortalLifecycleState = "disabled" | "enabling" | "enabled";

const ENABLE_DELAY_MS = 800;

// Prototype-only persistence; the real implementation reads portal_configurations.
export function usePortalLifecycle(resourceId: string) {
  const storageKey = `unkey:portal-enabled:${resourceId}`;

  const [state, setState] = useState<PortalLifecycleState>("disabled");
  const [hydrated, setHydrated] = useState(false);
  const timerRef = useRef<number | undefined>(undefined);

  useEffect(() => {
    setState(localStorage.getItem(storageKey) === "1" ? "enabled" : "disabled");
    setHydrated(true);
  }, [storageKey]);

  useEffect(() => () => window.clearTimeout(timerRef.current), []);

  const enable = useCallback(() => {
    setState("enabling");
    timerRef.current = window.setTimeout(() => {
      localStorage.setItem(storageKey, "1");
      setState("enabled");
    }, ENABLE_DELAY_MS);
  }, [storageKey]);

  const disable = useCallback(() => {
    localStorage.removeItem(storageKey);
    setState("disabled");
  }, [storageKey]);

  // Debug-only: jump straight to a state and hold it (no auto-advance).
  const forceState = useCallback(
    (next: PortalLifecycleState) => {
      window.clearTimeout(timerRef.current);
      if (next === "enabled") {
        localStorage.setItem(storageKey, "1");
      } else {
        localStorage.removeItem(storageKey);
      }
      setState(next);
    },
    [storageKey],
  );

  return { state, hydrated, enable, disable, forceState };
}
