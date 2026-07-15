"use client";

export type Product = "api-management" | "deploy";

export const STORAGE_KEY = "selected-product";
export const DEFAULT_PRODUCT: Product = "api-management";
export const STORAGE_EVENT = "product-selection-changed";

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
