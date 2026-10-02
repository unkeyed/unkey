import { useQuery } from "@tanstack/react-query";
import type { Key } from "~/components/keys-table/schema/keys.schema";
import { listKeys } from "~/lib/portal-api";

export const keysListQueryKey = ["portal", "keys", "list"] as const;

const PAGE_SIZE = 100;
const MAX_PAGES = 20;

async function listAllKeys(): Promise<{ keys: Key[]; truncated: boolean }> {
  const keys: Key[] = [];
  let cursor: string | undefined;
  for (let page = 0; page < MAX_PAGES; page += 1) {
    const result = await listKeys({ data: { cursor, limit: PAGE_SIZE } });
    keys.push(...result.keys);
    if (!result.hasMore || !result.cursor) {
      return { keys, truncated: false };
    }
    cursor = result.cursor;
  }
  return { keys, truncated: true };
}

export function useKeysListQuery() {
  const query = useQuery({
    queryKey: keysListQueryKey,
    queryFn: listAllKeys,
    staleTime: 1000 * 60,
    refetchOnWindowFocus: false,
  });

  return {
    keys: query.data?.keys ?? [],
    truncated: query.data?.truncated ?? false,
    isInitialLoading: query.isPending,
    isFetching: query.isFetching,
    isError: query.isError,
    error: query.error,
    refetch: query.refetch,
  };
}
