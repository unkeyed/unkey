/**
 * Whether a session may read keys. The portal keys page lists keys via
 * `portal.listKeys`, which the API authorizes with `read_key`, so the page must
 * only render for sessions granted the `keys:read` scope.
 *
 * Scopes use the portal's colon vocabulary, issued by `portal.createSession`
 * and persisted on the session (e.g. `keys:read`, `keys:reroll`).
 */
export function canReadKeys(scopes: ReadonlyArray<string>): boolean {
  return scopes.includes("keys:read");
}

/**
 * Whether a session may reroll a key it owns. `portal.rerollKey` authorizes
 * `keys:reroll`, so the table must not offer an action the API would refuse.
 */
export function canRerollKeys(scopes: ReadonlyArray<string>): boolean {
  return scopes.includes("keys:reroll");
}

/**
 * Whether a session may read verification analytics. `portal.getVerifications`
 * authorizes `read_analytics`, so the analytics page and its tab must only
 * appear for sessions granted `analytics:read`.
 */
export function canReadAnalytics(scopes: ReadonlyArray<string>): boolean {
  return scopes.includes("analytics:read");
}

export type PortalTab = {
  id: string;
  label: string;
  href: string;
};

/**
 * The header tabs a session can reach, in navigation order. Each tab is gated
 * on the scope its page's API call needs, so navigation never offers a page the
 * API would refuse to fill.
 */
export function deriveVisibleTabs(scopes: ReadonlyArray<string>): PortalTab[] {
  const tabs: PortalTab[] = [];
  if (canReadKeys(scopes)) {
    tabs.push({ id: "keys", label: "API keys", href: "/keys" });
  }
  if (canReadAnalytics(scopes)) {
    tabs.push({ id: "analytics", label: "Analytics", href: "/analytics" });
  }
  return tabs;
}

/**
 * Landing destination after session exchange: the first tab the session can
 * reach. Returns null when it can reach none, so the caller can surface an
 * appropriate state.
 */
export function getDefaultTabHref(scopes: ReadonlyArray<string>): string | null {
  return deriveVisibleTabs(scopes)[0]?.href ?? null;
}
