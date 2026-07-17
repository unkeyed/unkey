import type { PortalCapability } from "./capabilities";

/**
 * Portal tab identifiers. Each tab maps to a route in the portal app.
 */
export type PortalTab = "keys" | "analytics" | "docs";

type TabConfig = {
  id: PortalTab;
  label: string;
  href: string;
};

const TAB_CONFIGS: ReadonlyArray<TabConfig> = [
  { id: "keys", label: "API Keys", href: "/keys" },
  { id: "analytics", label: "Analytics", href: "/analytics" },
  { id: "docs", label: "Documentation", href: "/docs" },
] as const;

const KEY_CAPABILITIES: ReadonlySet<PortalCapability> = new Set([
  "keys:read",
  "keys:create",
  "keys:reroll",
]);

/**
 * Derive visible portal tabs from a session's capabilities.
 */
export function deriveVisibleTabs(
  permissions: ReadonlyArray<PortalCapability>,
): ReadonlyArray<TabConfig> {
  const hasKeys = permissions.some((permission) => KEY_CAPABILITIES.has(permission));
  const hasAnalytics = permissions.includes("analytics:read");
  const hasDocs = permissions.length > 0;

  return TAB_CONFIGS.filter((tab) => {
    switch (tab.id) {
      case "keys":
        return hasKeys;
      case "analytics":
        return hasAnalytics;
      case "docs":
        return hasDocs;
    }
  });
}

/**
 * Get the first visible tab's href for redirect after session exchange.
 */
export function getDefaultTabHref(permissions: ReadonlyArray<PortalCapability>): string | null {
  const tabs = deriveVisibleTabs(permissions);
  return tabs.length > 0 ? tabs[0].href : null;
}
