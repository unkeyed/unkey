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
 * Landing destination after session exchange.
 *
 * The portal currently exposes only the Keys page; Analytics and Docs are
 * deferred to v2 and blocked at the route layer. Returns null when the session
 * can't read keys so the caller can surface an appropriate state.
 */
export function getDefaultTabHref(scopes: ReadonlyArray<string>): string | null {
  return canReadKeys(scopes) ? "/keys" : null;
}
