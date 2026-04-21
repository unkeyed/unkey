"use client";

import { useCallback, useEffect, useMemo, useState } from "react";

export type NavbarVariant = "current" | "v1a" | "v1b" | "v2" | "v3";

export const STORAGE_KEY = "navbar-variant";
export const DEFAULT_VARIANT: NavbarVariant = "current";
export const STORAGE_EVENT = "navbar-variant-changed";

function isVariant(value: string | null): value is NavbarVariant {
  return (
    value === "current" || value === "v1a" || value === "v1b" || value === "v2" || value === "v3"
  );
}

/**
 * Dev-only prototype switch between navbar variants. Mirrors
 * `useProductSelection`: lazy-init from localStorage, custom event
 * for cross-component sync, no `any` / `!` / `as`.
 */
export function useNavbarVariant() {
  const [variant, setVariantState] = useState<NavbarVariant>(() => {
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

  const setVariant = useCallback((next: NavbarVariant) => {
    setVariantState(next);
    if (typeof window !== "undefined") {
      localStorage.setItem(STORAGE_KEY, next);
      window.dispatchEvent(new Event(STORAGE_EVENT));
    }
  }, []);

  return useMemo(() => ({ variant, setVariant }), [variant, setVariant]);
}

/**
 * Read the current variant without React state. Useful for layout-level
 * code that needs to gate rendering without subscribing to updates.
 */
export function getCurrentVariant(): NavbarVariant {
  if (typeof window === "undefined") {
    return DEFAULT_VARIANT;
  }
  const saved = localStorage.getItem(STORAGE_KEY);
  return isVariant(saved) ? saved : DEFAULT_VARIANT;
}
