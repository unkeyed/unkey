import { useEffect } from "react";

type UsePageClampArgs = {
  /** Current 1-based page from URL state. */
  page: number;
  /** Total pages derived from the latest settled result (min 1). */
  totalPages: number;
  /** True while a query for new params is in flight. */
  isFetching: boolean;
  /** True once the query has produced a result at least once. */
  hasData: boolean;
  setPage: (page: number) => void;
};

// usePageClamp snaps `page` back into [1, totalPages] once the current query
// has settled. It gates on !isFetching because paginated list queries use
// keepPreviousData: while a filter/sort/time change is in flight, `data` (and
// thus totalPages) still reflects the previous query, so clamping mid-fetch
// would race the reset-to-page-1 effect and land on a page derived from the
// old result set. The hasData guard keeps a deep-linked page (e.g. ?page=7)
// from snapping to 1 before the first result loads, when totalPages still
// collapses to 1.
export function usePageClamp({ page, totalPages, isFetching, hasData, setPage }: UsePageClampArgs) {
  useEffect(() => {
    if (isFetching || !hasData) {
      return;
    }
    if (page > totalPages) {
      setPage(totalPages);
    }
  }, [isFetching, hasData, page, totalPages, setPage]);
}
