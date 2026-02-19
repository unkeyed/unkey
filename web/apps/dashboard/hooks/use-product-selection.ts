"use client";

import { useRouter } from "next/navigation";
import { useSelectedLayoutSegments } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { useWorkspaceNavigation } from "./use-workspace-navigation";

export type Product = "api-management" | "deploy";

const STORAGE_KEY = "selected-product";
const DEFAULT_PRODUCT: Product = "api-management";

// Product-specific route patterns
const PRODUCT_ROUTES = {
  "api-management": ["apis", "ratelimits", "authorization", "logs", "identities"],
  deploy: ["projects", "domains", "environment-variables"],
};

/**
 * Hook to manage product selection state with localStorage persistence.
 *
 * Features:
 * - Persists product selection across sessions
 * - Defaults to last used product, or API Management if none exists
 * - Handles navigation when switching products
 * - Updates product selection when navigating to product-specific routes
 */
export function useProductSelection() {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const segments = useSelectedLayoutSegments();

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

  // Update product selection when navigating to product-specific routes
  useEffect(() => {
    if (!segments || segments.length < 2) {
      return;
    }

    const productSegment = segments[1];

    // Check if we're on a Deploy route
    if (PRODUCT_ROUTES.deploy.some((route) => productSegment?.startsWith(route))) {
      if (product !== "deploy") {
        setProductState("deploy");
        if (typeof window !== "undefined") {
          localStorage.setItem(STORAGE_KEY, "deploy");
        }
      }
      return;
    }

    // Check if we're on an API Management route
    if (PRODUCT_ROUTES["api-management"].some((route) => productSegment?.startsWith(route))) {
      if (product !== "api-management") {
        setProductState("api-management");
        if (typeof window !== "undefined") {
          localStorage.setItem(STORAGE_KEY, "api-management");
        }
      }
    }
  }, [segments, product]);

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
