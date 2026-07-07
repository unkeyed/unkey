import { useEffect } from "react";

type UsePageClampArgs = {
  /**
   * The transition-aware page from usePageTransition — never the raw URL
   * page. With keepPreviousData, `data` (and thus totalPages) reflects the
   * previous query while a filter/search/time change is in flight, and
   * clamping the outgoing page against those totals would race the
   * reset-to-page-1 effect — including on cache hits, where no fetch state
   * ever flips. The transition-aware page is 1 during that window, which
   * makes the clamp a no-op.
   */
  page: number;
  /** Total pages derived from the latest result (min 1). */
  totalPages: number;
  /** Latest query result, passed as-is — nullish means no result yet. */
  data: unknown;
  setPage: (page: number) => void;
};

// usePageClamp snaps `page` back into [1, totalPages] once the query has a
// result. The data guard keeps a deep-linked page (?page=7) from snapping to
// 1 before the first response, when totalPages still collapses to 1. The
// clamp is deliberately eager — no isFetching gate — so an out-of-range page
// within an unchanged result set (e.g. a hand-edited ?page=999) snaps back
// instantly using the already-known totals instead of waiting for the dead
// page's request to settle or exhaust retries.
export function usePageClamp({ page, totalPages, data, setPage }: UsePageClampArgs) {
  useEffect(() => {
    if (data == null) {
      return;
    }
    if (page > totalPages) {
      setPage(totalPages);
    }
  }, [data, page, totalPages, setPage]);
}
