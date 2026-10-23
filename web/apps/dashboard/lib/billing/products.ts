export const BILLING_PRODUCTS = ["compute", "api"] as const;

/**
 * Each product owns a whole Stripe subscription, tracked as one row per
 * (workspace, product) in billing_subscriptions.
 */
export type BillingProduct = (typeof BILLING_PRODUCTS)[number];

export const PRODUCT_LABELS: Record<BillingProduct, string> = {
  compute: "Compute",
  api: "API",
};
