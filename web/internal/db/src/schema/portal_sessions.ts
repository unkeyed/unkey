import { relations } from "drizzle-orm";
import { bigint, boolean, index, json, mysqlTable, uniqueIndex } from "drizzle-orm/mysql-core";
import { portals } from "./portals";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";
import { workspaces } from "./workspaces";

/**
 * A portal session covers the whole life of one end-user visit: the short-lived
 * exchange code `portal.createSession` mints, and the long-lived access token
 * `portal.exchangeCode` swaps it for. The relationship is 1:1 by construction —
 * a code must never mint more than one access token — so the two live on one
 * row rather than in two tables with no FK between them.
 *
 * Both credentials are stored only as hashes (same digest and encoding as
 * `keys.hash`), each behind a UNIQUE index so a lookup stays one indexed read.
 * `id` is the non-secret row handle used by audit logs and API responses.
 *
 * Single-use redemption is structural rather than enforced in code: the
 * exchange updates the row `WHERE exchange_code_hash = ? AND access_token_hash
 * IS NULL AND exchange_code_expires_at > ?`, so concurrent redemptions race on
 * one row and no write path can forget the check.
 *
 * State is derived, never stored — two of the five states are clock-driven, so
 * a status column would go stale with no write to trigger it. Vitess cannot
 * enforce the legal combinations, so they are documented here and asserted in
 * `internal/services/portal`:
 *
 *   pending       exchange_code_hash set, access_token_hash NULL, code not expired
 *   code_expired  exchange_code_hash set, access_token_hash NULL, code expired unused
 *   active        access_token_hash set, not expired, not revoked
 *   expired       access_token_hash set, access_token_expires_at passed
 *   revoked       revoked_at set (takes precedence over expired, and over corrupt)
 *
 * `access_token_hash` set with either `access_token_created_at` or
 * `access_token_expires_at` NULL is corrupt, not a state. A missing expiry is
 * read as expired rather than active so a malformed row can never authenticate,
 * and the reader reports the corruption separately.
 */
export const portalSessions = mysqlTable(
  "portal_sessions",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    portalId: id("portal_id").notNull(),
    externalId: caseSensitiveVarchar("external_id", { length: 256 }).notNull(),
    scopes: json("scopes").notNull(),
    preview: boolean("preview").notNull().default(false),

    exchangeCodeHash: caseSensitiveVarchar("exchange_code_hash", { length: 256 }).notNull(),
    exchangeCodeExpiresAt: bigint("exchange_code_expires_at", { mode: "number" }).notNull(),

    accessTokenHash: caseSensitiveVarchar("access_token_hash", { length: 256 }),
    accessTokenCreatedAt: bigint("access_token_created_at", { mode: "number" }),
    accessTokenExpiresAt: bigint("access_token_expires_at", { mode: "number" }),

    revokedAt: bigint("revoked_at", { mode: "number" }),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
  },
  (table) => [
    uniqueIndex("idx_exchange_code_hash").on(table.exchangeCodeHash),
    uniqueIndex("idx_access_token_hash").on(table.accessTokenHash),
    index("idx_workspace").on(table.workspaceId),
    index("idx_external_id").on(table.externalId),
    index("idx_exchange_code_expires").on(table.exchangeCodeExpiresAt),
    index("idx_access_token_expires").on(table.accessTokenExpiresAt),
  ],
);

export const portalSessionsRelations = relations(portalSessions, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [portalSessions.workspaceId],
    references: [workspaces.id],
  }),
  portal: one(portals, {
    fields: [portalSessions.portalId],
    references: [portals.id],
  }),
}));
