import { getUnkeyClient } from "@/lib/unkey-client";
import type { V2RootKeysListKeysResponseData } from "@unkey/api/models/components";

export type V2RootKey = V2RootKeysListKeysResponseData;

export const rootKeysV2QueryKeys = {
  all: ["rootKeys", "v2"] as const,
  workspace: (workspaceId: string) => [...rootKeysV2QueryKeys.all, workspaceId] as const,
};

export async function listRootKeys(signal?: AbortSignal): Promise<V2RootKey[]> {
  const rootKeys: V2RootKey[] = [];
  const cursors = new Set<string>();
  const pages = await getUnkeyClient().rootKeys.listKeys({ limit: 100 }, { signal });

  for await (const { result: page } of pages) {
    rootKeys.push(...page.data);

    const { cursor, hasMore } = page.pagination;
    if (!hasMore) {
      break;
    }
    if (!cursor) {
      throw new Error("Root keys API returned another page without a cursor");
    }
    if (cursors.has(cursor)) {
      throw new Error("Root keys API returned a repeated cursor");
    }
    cursors.add(cursor);
  }

  return rootKeys;
}
