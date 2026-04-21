/**
 * Single source of truth for the API → Keyspace rename.
 *
 * Soft-transition window: nav uses the long form so the team's existing
 * customers see the rename in flight. Once support tickets stop mentioning
 * "APIs" we drop the parenthetical by editing this one file.
 *
 * - Use {@link KEYSPACE_LABEL} in nav items.
 * - Use {@link KEYSPACE_LABEL_SHORT} in breadcrumbs and page headers
 *   (the user is already inside the resource — no need to repeat the
 *   "(formerly APIs)" annotation).
 */
export const KEYSPACE_LABEL = "Keyspaces (APIs)";
export const KEYSPACE_LABEL_SHORT = "Keyspaces";
