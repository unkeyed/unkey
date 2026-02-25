"use client";

import { useRouter } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useWorkspaceNavigation } from "./use-workspace-navigation";

export type Product = "api-management" | "deploy";

const STORAGE_KEY = "selected-product";
const DEFAULT_PRODUCT: Product = "api-management";
const STORAGE_EVENT = "product-selection-changed";

const PRODUCT_HOME_ROUTES: Record<Product, string> = {
  "api-management": "apis",
  deploy: "projects",
};

/**
 * Hook to manage product selection state with localStorage persistence.
 *
 * Features:
 * - Persists product selection across sessions
 * - Defaults to last used product, or API Management if none exists
 * - Handles navigation when switching products
 * - Product selection is explicit via switchProduct() only
 * - Syncs across components via custom event
 */
export function useProductSelection() {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();

  // Initialize state with localStorage value (lazy initialization)
  const [product, setProductState] = useState<Product>(() => {
    if (typeof window === "undefined") {
      return DEFAULT_PRODUCT;
    }

    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved === "api-management" || saved === "deploy") {
      return saved;
    }
    return DEFAULT_PRODUCT;
  });

  // Listen for product changes from other components
  useEffect(() => {
    const handleStorageChange = () => {
      const saved = localStorage.getItem(STORAGE_KEY);
      if (saved === "api-management" || saved === "deploy") {
        setProductState(saved);
      }
    };

    window.addEventListener(STORAGE_EVENT, handleStorageChange);
    return () => window.removeEventListener(STORAGE_EVENT, handleStorageChange);
  }, []);

  const switchProduct = useCallback(
    (newProduct: Product) => {
      setProductState(newProduct);

      // Persist to localStorage
      if (typeof window !== "undefined") {
        localStorage.setItem(STORAGE_KEY, newProduct);
        // Dispatch custom event to notify other components
        window.dispatchEvent(new Event(STORAGE_EVENT));
      }

      // Navigate to product home
      router.push(`/${workspace.slug}/${PRODUCT_HOME_ROUTES[newProduct]}`);
    },
    [router, workspace.slug],
  );

  return useMemo(
    () => ({
      product,
      switchProduct,
    }),
    [product, switchProduct],
  );
}

/**
 * Get the current product from localStorage without React state.
 * Used by useNavigationContext to read the current selection.
 */
export function getCurrentProduct(): Product {
  if (typeof window === "undefined") {
    return DEFAULT_PRODUCT;
  }

  const saved = localStorage.getItem(STORAGE_KEY);
  if (saved === "api-management" || saved === "deploy") {
    return saved;
  }
  return DEFAULT_PRODUCT;
}
