"use client";

import { getUnkeyClient } from "@/lib/unkey-client";
import type { Unkey } from "@unkey/api";

type GetPortalResult = Awaited<ReturnType<Unkey["portal"]["getPortal"]>>["data"];

/**
 * Reads the portal serving a keyspace.
 *
 * `getPortal` takes an undiscriminated union of every way to name a portal, so
 * TypeScript accepts several keys at once and only the server rejects it.
 * Narrowing to one keyspace id here makes that shape unconstructible at the
 * call site.
 */
export async function getPortalByKeyspace(keyspaceId: string): Promise<GetPortalResult> {
  const response = await getUnkeyClient().portal.getPortal({ keyspaceId });
  return response.data;
}
