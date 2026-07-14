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
 * Analytics capability that grants visibility to the Analytics tab.
 */
const ANALYTICS_READ = "analytics:read";

/**
 * Derive visible portal tabs from a session's capabilities.
 *
 * Capabilities use the portal's colon vocabulary, issued by
 * `portal.createSession` and persisted on the session
 * (e.g. `keys:read`, `keys:reroll`, `analytics:read`). Per the RFC:
 * - Keys tab: any `keys:*` capability
 * - Analytics tab: `analytics:read`
 * - Docs tab: visible when any capability is present
 */
export function deriveVisibleTabs(permissions: ReadonlyArray<string>): ReadonlyArray<TabConfig> {
  const hasKeys = permissions.some((p) => p.startsWith("keys:"));
  const hasAnalytics = permissions.includes(ANALYTICS_READ);
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
 * Whether a session may read keys. The portal keys page lists keys via
 * `portal.listKeys`, which the API authorizes with `read_key`, so the page must
 * only render for sessions granted the `keys:read` capability.
 */
export function canReadKeys(permissions: ReadonlyArray<string>): boolean {
  return permissions.includes("keys:read");
}

/**
 * Get the first visible tab's href for redirect after session exchange.
 */
export function getDefaultTabHref(permissions: ReadonlyArray<string>): string | null {
  const tabs = deriveVisibleTabs(permissions);
  return tabs.length > 0 ? tabs[0].href : null;
}
