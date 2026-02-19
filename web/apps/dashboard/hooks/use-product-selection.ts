"use client";

import { useRouter } from "next/navigation";
import { useCallback, useState } from "react";
import { useWorkspaceNavigation } from "./use-workspace-navigation";

export type Product = "api-management" | "deploy";

const STORAGE_KEY = "selected-product";
const DEFAULT_PRODUCT: Product = "api-management";

/**
 * Hook to manage product selection state with localStorage persistence.
 *
 * Features:
 * - Persists product selection across sessions
 * - Defaults to last used product, or API Management if none exists
 * - Handles navigation when switching products
 */
export function useProductSelection() {
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

  const router = useRouter();
  const workspace = useWorkspaceNavigation();

  const switchProduct = useCallback(
    (newProduct: Product) => {
      setProductState(newProduct);

      // Persist to localStorage
      if (typeof window !== "undefined") {
        localStorage.setItem(STORAGE_KEY, newProduct);
      }

      // Navigate to product home
      if (newProduct === "api-management") {
        router.push(`/${workspace.slug}/apis`);
      } else {
        router.push(`/${workspace.slug}/projects`);
      }
    },
    [router, workspace.slug],
  );

  return {
    product,
    switchProduct,
  };
}
