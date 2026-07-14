import { z } from "zod";

/**
 * A portal-facing API key row. This is the shape the keys-table components
 * render; it is mapped from the `v2/portal.listKeys` response (see
 * {@link mapListKeysResponse}) and kept intentionally small — the portal only
 * ever reads and rerolls keys, so fields the end user cannot change (ratelimits,
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

// Only the fields the portal keys table needs. The endpoint returns the shared
// KeyResponseData shape (permissions, roles, credits, ratelimits, …); we ignore
// the rest rather than model fields the portal never surfaces.
const keyResponseSchema = z.object({
  keyId: z.string(),
  name: z.string().nullish(),
  start: z.string(),
  createdAt: z.number(),
  expires: z.number().nullish(),
  enabled: z.boolean(),
});

export const listKeysResponseSchema = z.object({
  data: z.array(keyResponseSchema),
  pagination: z
    .object({
      cursor: z.string().nullish(),
      hasMore: z.boolean(),
    })
    .nullish(),
});

/**
 * Map a validated `v2/portal.listKeys` response into the table's {@link Key}
 * shape.
 */
export function mapListKeysResponse(parsed: z.infer<typeof listKeysResponseSchema>): KeysPage {
  return {
    keys: parsed.data.map((k) => ({
      id: k.keyId,
      name: k.name ?? null,
      start: k.start,
      createdAt: k.createdAt,
      expires: k.expires ?? null,
      enabled: k.enabled,
      usage: [],
    })),
    cursor: parsed.pagination?.cursor ?? null,
    hasMore: parsed.pagination?.hasMore ?? false,
  };
}

/** Query params for one page of `v2/portal.listKeys`. */
export type ListKeysQuery = {
  cursor?: string;
  limit?: number;
};

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

export const rerollKeyResponseSchema = z.object({
  data: z.object({
    // The plaintext secret — shown to the user exactly once.
    key: z.string(),
    keyId: z.string(),
  }),
});

/** The one-time secret returned by a successful reroll. */
export type RerollKeyResult = {
  keyId: string;
  plaintext: string;
};
