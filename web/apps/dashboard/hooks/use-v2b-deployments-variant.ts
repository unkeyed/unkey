"use client";

import { useCallback, useEffect, useMemo, useState } from "react";

export type V2bDeploymentsVariant = "a" | "b" | "c" | "d" | "e" | "f";

export const STORAGE_KEY = "v2b-deployments-variant";
export const DEFAULT_VARIANT: V2bDeploymentsVariant = "a";
export const STORAGE_EVENT = "v2b-deployments-variant-changed";

function isVariant(value: string | null): value is V2bDeploymentsVariant {
  return (
    value === "a" ||
    value === "b" ||
    value === "c" ||
    value === "d" ||
    value === "e" ||
    value === "f"
  );
}

/**
 * Dev-only sub-variant axis layered on top of the v2b navbar variant.
 * Mirrors `useNavbarVariant`: lazy-init from localStorage, custom
 * event for cross-component sync. Only consulted by the deployments
 * shell + the v2b top-header crumb extension.
 */
export function useV2bDeploymentsVariant() {
  const [variant, setVariantState] = useState<V2bDeploymentsVariant>(() => {
    if (typeof window === "undefined") {
      return DEFAULT_VARIANT;
    }
    const saved = localStorage.getItem(STORAGE_KEY);
    return isVariant(saved) ? saved : DEFAULT_VARIANT;
  });

  useEffect(() => {
    const handleChange = () => {
      const saved = localStorage.getItem(STORAGE_KEY);
      if (isVariant(saved)) {
        setVariantState(saved);
      }
    };
    window.addEventListener(STORAGE_EVENT, handleChange);
    return () => window.removeEventListener(STORAGE_EVENT, handleChange);
  }, []);

  const setVariant = useCallback((next: V2bDeploymentsVariant) => {
    setVariantState(next);
    if (typeof window !== "undefined") {
      localStorage.setItem(STORAGE_KEY, next);
      window.dispatchEvent(new Event(STORAGE_EVENT));
    }
  }, []);

  return useMemo(() => ({ variant, setVariant }), [variant, setVariant]);
}

export function getCurrentV2bDeploymentsVariant(): V2bDeploymentsVariant {
  if (typeof window === "undefined") {
    return DEFAULT_VARIANT;
  }
  const saved = localStorage.getItem(STORAGE_KEY);
  return isVariant(saved) ? saved : DEFAULT_VARIANT;
}
