import { type QueryKey, useQueryClient } from "@tanstack/react-query";
import { flushSync } from "react-dom";

export function useAnchoredRefresh(anchor: () => void, queryKeys: readonly QueryKey[]) {
  const queryClient = useQueryClient();

  return () => {
    const activeBeforeAnchor = new Set(
      queryKeys.flatMap((queryKey) =>
        queryClient
          .getQueryCache()
          .findAll({ queryKey, type: "active" })
          .map((query) => query.queryHash),
      ),
    );
    flushSync(() => {
      anchor();
    });
    for (const queryKey of queryKeys) {
      void queryClient.invalidateQueries(
        { queryKey, predicate: (query) => !activeBeforeAnchor.has(query.queryHash) },
        { cancelRefetch: false },
      );
      void queryClient.invalidateQueries({
        queryKey,
        predicate: (query) => activeBeforeAnchor.has(query.queryHash),
      });
    }
  };
}
