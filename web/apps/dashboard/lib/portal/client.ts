"use client";

import { getUnkeyClient } from "@/lib/unkey-client";
import type { Unkey } from "@unkey/api";
import type { PortalMapping } from "@unkey/api/models/components";

type GetPortalResult = Awaited<ReturnType<Unkey["portal"]["getPortal"]>>["data"];

/** The keyspace mapping a per-API portal surface addresses. */
export function keyspaceMapping(keyAuthId: string): PortalMapping {
  return { type: "keyspace", id: keyAuthId };
}

/**
 * Reads a portal by the resource it serves.
 *
 * `getPortal` takes a union of "name the portal" and "name the mapping" that is
 * not discriminated, so TypeScript happily accepts both keys at once — a shape
 * the server rejects with a 400. Going through this helper makes that shape
 * unconstructible at the call site.
 */
export async function getPortalByMapping(mapping: PortalMapping): Promise<GetPortalResult> {
  const response = await getUnkeyClient().portal.getPortal({ mapping });
  return response.data;
}
