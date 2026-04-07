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

/**
 * Derive visible portal tabs from a session's permissions array.
 *
 * - Any permission matching `keys:*` → Keys tab visible
 * - Any permission matching `analytics:*` → Analytics tab visible
 * - `docs:read` → Docs tab visible
 */
export function deriveVisibleTabs(permissions: ReadonlyArray<string>): ReadonlyArray<TabConfig> {
  const hasKeys = permissions.some((p) => p.startsWith("keys:"));
  const hasAnalytics = permissions.some((p) => p.startsWith("analytics:"));
  const hasDocs = permissions.includes("docs:read");

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
export function getDefaultTabHref(permissions: ReadonlyArray<string>): string | null {
  const tabs = deriveVisibleTabs(permissions);
  return tabs.length > 0 ? tabs[0].href : null;
}
