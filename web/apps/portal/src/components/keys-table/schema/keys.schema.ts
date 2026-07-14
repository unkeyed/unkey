import { z } from "zod";

/**
 * A portal-facing API key row. This is the shape the keys-table components
 * render; it is mapped from the `v2/portal.listKeys` SDK response in
 * `lib/portal-api.ts` and kept intentionally small — the portal only ever reads
 * and rerolls keys, so fields the end user cannot change (ratelimits,
 * permissions, roles, metadata) are deliberately omitted.
 */
export type Key = {
  id: string;
  name: string | null;
  start: string;
  createdAt: number;
  expires: number | null;
  enabled: boolean;
  /** Daily verification buckets, oldest → newest. Empty until analytics wire-up. */
  usage: number[];
  errors?: number[];
};

/** A single page of keys plus the cursor to fetch the next one. */
export type KeysPage = {
  keys: Key[];
  cursor: string | null;
  hasMore: boolean;
};

/** Query params for one page of `v2/portal.listKeys`. */
export const listKeysQuerySchema = z.object({
  cursor: z.string().optional(),
  limit: z.number().int().positive().max(100).optional(),
});

export type ListKeysQuery = z.infer<typeof listKeysQuerySchema>;

/** Request body for `v2/portal.rerollKey`. */
export const rerollKeyRequestSchema = z.object({
  keyId: z.string().min(1),
  /**
   * Duration in milliseconds until the ORIGINAL key is revoked, starting now.
   * `0` revokes immediately; positive values keep it valid for a grace period.
   */
  expiration: z.number().int().nonnegative(),
});

export type RerollKeyRequest = z.infer<typeof rerollKeyRequestSchema>;

/** The one-time secret returned by a successful reroll. */
export type RerollKeyResult = {
  keyId: string;
  plaintext: string;
};
