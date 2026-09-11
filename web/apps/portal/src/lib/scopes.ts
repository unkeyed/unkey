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
 * authorizes `read_analytics`, so the keys page must only render its metrics,
 * chart and per-key request counts for sessions granted `analytics:read`.
 */
export function canReadAnalytics(scopes: ReadonlyArray<string>): boolean {
  return scopes.includes("analytics:read");
}

/**
 * Landing destination after session exchange. Null when the session can reach
 * no page, so the caller can surface an appropriate state.
 */
export function getDefaultTabHref(scopes: ReadonlyArray<string>): string | null {
  return canReadKeys(scopes) ? "/keys" : null;
}
