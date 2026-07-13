import { useEffect, useRef } from "react";

type UsePageTransitionArgs = {
  /**
   * Stable string key of everything that invalidates the current page — the
   * OFFSET is only meaningful relative to the query that produced it, so the
   * key must cover filters, search, and the time window. Sort order does not
   * change totals, so it is safe to omit — but then a sort handler that wants
   * pagination reset must call setPage(1) itself (see onSortingChange in the
   * table hooks that do this).
   */
  transitionKey: string;
  /** Current 1-based page from URL state. */
  page: number;
  setPage: (page: number) => void;
  /** Extra reset work tied to the same transition (e.g. clearing a realtime buffer). */
  onTransition?: () => void;
};

// usePageTransition resets pagination to page 1 when transitionKey changes
// and returns the page the query should use. The returned page is forced to 1
// synchronously on the render that first observes the change, before the
// setPage(1) effect commits the URL: querying with the outgoing page would
// fire (and cache forever, given staleTime: Infinity) a request for a page
// that may not exist under the new key, and downstream effects like
// usePageClamp must never compare the outgoing page against the new result's
// totals. The ref starts null so a URL-persisted page (?page=7 deep link)
// survives first mount.
export function usePageTransition({
  transitionKey,
  page,
  setPage,
  onTransition,
}: UsePageTransitionArgs): number {
  const prevKeyRef = useRef<string | null>(null);
  const isTransitioning = prevKeyRef.current !== null && prevKeyRef.current !== transitionKey;

  const onTransitionRef = useRef(onTransition);
  onTransitionRef.current = onTransition;

  useEffect(() => {
    if (prevKeyRef.current === null) {
      prevKeyRef.current = transitionKey;
      return;
    }
    if (transitionKey !== prevKeyRef.current) {
      prevKeyRef.current = transitionKey;
      setPage(1);
      onTransitionRef.current?.();
    }
  }, [transitionKey, setPage]);

  return isTransitioning ? 1 : page;
}
