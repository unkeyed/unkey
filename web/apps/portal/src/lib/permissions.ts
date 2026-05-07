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
 * RBAC actions that grant visibility to the Keys tab.
 */
const KEY_ACTIONS = new Set(["read_key", "create_key", "update_key", "delete_key"]);

/**
 * RBAC actions that grant visibility to the Analytics tab.
 */
const ANALYTICS_ACTIONS = new Set(["read_analytics"]);

/**
 * Derive visible portal tabs from a session's RBAC tuple permissions.
 *
 * Each permission is expected in the format `{resourceType}.{resourceId}.{action}`.
 * The action segment (third dot-separated segment) determines tab visibility:
 * - Keys tab: action ∈ {read_key, create_key, update_key, delete_key}
 * - Analytics tab: action = read_analytics
 * - Docs tab: visible when any permission is present (regardless of action)
 *
 * Permissions with fewer than 3 segments are silently ignored (defensive fallback).
 */
export function deriveVisibleTabs(permissions: ReadonlyArray<string>): ReadonlyArray<TabConfig> {
  const actions = permissions
    .map((p) => {
      const parts = p.split(".");
      return parts.length === 3 ? parts[2] : null;
    })
    .filter((a): a is string => a !== null);

  const hasKeys = actions.some((a) => KEY_ACTIONS.has(a));
  const hasAnalytics = actions.some((a) => ANALYTICS_ACTIONS.has(a));
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
export function getDefaultTabHref(permissions: ReadonlyArray<string>): string | null {
  const tabs = deriveVisibleTabs(permissions);
  return tabs.length > 0 ? tabs[0].href : null;
}
