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
  let cursor: string | undefined;

  do {
    const { result: page } = await getUnkeyClient().rootKeys.listKeys(
      { limit: 100, cursor },
      { signal },
    );
    rootKeys.push(...page.data);

    if (page.pagination.hasMore && !page.pagination.cursor) {
      throw new Error("Root keys API returned another page without a cursor");
    }
    if (page.pagination.cursor && cursors.has(page.pagination.cursor)) {
      throw new Error("Root keys API returned a repeated cursor");
    }
    if (page.pagination.cursor) {
      cursors.add(page.pagination.cursor);
    }
    cursor = page.pagination.hasMore ? page.pagination.cursor : undefined;
  } while (cursor);

  return rootKeys;
}
