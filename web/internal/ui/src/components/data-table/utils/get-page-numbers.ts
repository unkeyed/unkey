/**
 * Generates an array of exactly 7 page slots (or fewer when totalPages ≤ 7)
 * for pagination UI. When only one ellipsis is needed, extra page numbers
 * fill the remaining slots so the output length stays constant.
 *
 * @param page        - Current page (1-indexed)
 * @param totalPages  - Total number of pages
 */
export function getPageNumbers(page: number, totalPages: number): Array<number | "ellipsis"> {
  const TOTAL_SLOTS = 7;

  if (totalPages <= TOTAL_SLOTS) {
    return Array.from({ length: totalPages }, (_, i) => i + 1);
  }

  // Near start — no leading ellipsis, show first 5 pages + trailing ellipsis + last
  if (page < 5) {
    return [1, 2, 3, 4, 5, "ellipsis", totalPages];
  }

  // Near end — leading ellipsis + last 5 pages
  if (page > totalPages - 4) {
    return [
      1,
      "ellipsis",
      totalPages - 4,
      totalPages - 3,
      totalPages - 2,
      totalPages - 1,
      totalPages,
    ];
  }

  // Middle — both ellipses with a 3-page window around current
  return [1, "ellipsis", page - 1, page, page + 1, "ellipsis", totalPages];
}
