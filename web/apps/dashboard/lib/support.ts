import type { Route } from "next";

// typedRoutes only knows app-router paths, so an off-app destination has to be
// asserted. Done once here so call sites stay cast-free.
export const SUPPORT_MAILTO = "mailto:support@unkey.com" as Route;

export const BILLING_DOCS =
  "https://www.unkey.com/docs/platform/workspaces/billing/overview" as Route;
export const COMPUTE_BILLING_DOCS =
  "https://www.unkey.com/docs/platform/workspaces/billing/compute" as Route;
export const ROOT_KEY_IP_ALLOWLIST_DOCS =
  "https://www.unkey.com/docs/platform/root-keys/configure#restrict-a-root-key-by-ip-address" as Route;
