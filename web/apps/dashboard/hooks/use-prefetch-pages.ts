import { useCallback, useEffect, useRef } from "react";

const DEFAULT_PAGES_AHEAD = 2;

type UsePrefetchPagesArgs<TParams extends { page: number }> = {
  /** Transition-aware current page (queryPage), not the raw URL page. */
  page: number;
  /** Total pages derived from the latest result (min 1). */
  totalPages: number;
  /** Query params for the current page; each prefetch overrides only `page`. */
  queryParams: TParams;
  /**
   * Prefetch a single page. Callers pass a fresh arrow each render (usually a
   * tRPC `utils.*.prefetch` call baking in staleTime); it is stabilized via a
   * ref so a new identity does not re-fire the effect.
   */
  prefetch: (params: TParams) => void;
  /** How many pages ahead to warm. Defaults to 2. */
  pagesAhead?: number;
};

// usePrefetchPages warms the next few pages of a paginated query so forward
// navigation feels instant. It stops at totalPages and re-runs only when the
// page, totals, or query params actually change.
export function usePrefetchPages<TParams extends { page: number }>({
  page,
  totalPages,
  queryParams,
  prefetch,
  pagesAhead = DEFAULT_PAGES_AHEAD,
}: UsePrefetchPagesArgs<TParams>) {
  // Store-latest-callback: a fresh caller arrow each render must not re-fire the
  // effect. The ref keeps the effect keyed on real inputs only.
  const prefetchRef = useRef(prefetch);
  prefetchRef.current = prefetch;
  const prefetchPage = useCallback((params: TParams) => prefetchRef.current(params), []);

  useEffect(() => {
    for (let i = 1; i <= pagesAhead; i++) {
      const nextPage = page + i;
      if (nextPage > totalPages) {
        break;
      }
      prefetchPage({ ...queryParams, page: nextPage });
    }
  }, [page, totalPages, queryParams, pagesAhead, prefetchPage]);
}
