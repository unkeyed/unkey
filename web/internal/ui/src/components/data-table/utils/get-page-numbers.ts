/**
 * Generates an array of page numbers (with "ellipsis" markers) for pagination UI.
 * @param page - Current page (1-indexed)
 * @param totalPages - Total number of pages
 * @param maxVisible - Maximum number of page buttons to show (default 5)
 */
export function getPageNumbers(
  page: number,
  totalPages: number,
  maxVisible = 5,
): Array<number | "ellipsis"> {
  const pages: Array<number | "ellipsis"> = [];

  if (totalPages <= maxVisible) {
    for (let i = 1; i <= totalPages; i++) {
      pages.push(i);
    }
    return pages;
  }

  pages.push(1);

  if (page > 3) {
    pages.push("ellipsis");
  }

  const startPage = Math.max(2, page - 1);
  const endPage = Math.min(totalPages - 1, page + 1);

  for (let i = startPage; i <= endPage; i++) {
    pages.push(i);
  }

  if (page < totalPages - 2) {
    pages.push("ellipsis");
  }

  pages.push(totalPages);

  return pages;
}
