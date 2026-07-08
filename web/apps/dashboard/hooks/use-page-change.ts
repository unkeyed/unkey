import { useCallback } from "react";

// usePageChange returns a bounds-checked page setter for a paginated table.
// Out-of-range requests (below 1 or past the last page) are ignored rather than
// clamped, so a stale UI control cannot move the user to a page that does not
// exist. The returned callback is stable for a given (totalPages, setPage).
export function usePageChange(totalPages: number, setPage: (page: number) => void) {
  return useCallback(
    (newPage: number) => {
      if (newPage < 1 || newPage > totalPages) {
        return;
      }
      setPage(newPage);
    },
    [totalPages, setPage],
  );
}
