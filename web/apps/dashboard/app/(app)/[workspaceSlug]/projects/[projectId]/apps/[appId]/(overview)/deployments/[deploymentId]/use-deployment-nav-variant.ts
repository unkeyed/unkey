"use client";

import { useCallback, useEffect, useState } from "react";

export type DeploymentNavVariant = "breadcrumb" | "sidebar" | "crumb";

const STORAGE_KEY = "deployment-nav-variant";
const CHANGE_EVENT = "deployment-nav-variant-change";

function isVariant(value: string | null): value is DeploymentNavVariant {
  return value === "breadcrumb" || value === "sidebar" || value === "crumb";
}

/**
 * Temporary test toggle: switch the deployment detail between the breadcrumb +
 * horizontal tabs and the vertical SecondaryNav rail. Backed by localStorage
 * and synced across components via a window event. Remove once a variant wins.
 */
export function useDeploymentNavVariant(): [
  DeploymentNavVariant,
  (next: DeploymentNavVariant) => void,
] {
  const [variant, setVariantState] = useState<DeploymentNavVariant>("breadcrumb");

  useEffect(() => {
    const sync = () => {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (isVariant(stored)) {
        setVariantState(stored);
      }
    };
    sync();
    window.addEventListener(CHANGE_EVENT, sync);
    return () => window.removeEventListener(CHANGE_EVENT, sync);
  }, []);

  const setVariant = useCallback((next: DeploymentNavVariant) => {
    localStorage.setItem(STORAGE_KEY, next);
    setVariantState(next);
    window.dispatchEvent(new Event(CHANGE_EVENT));
  }, []);

  return [variant, setVariant];
}
