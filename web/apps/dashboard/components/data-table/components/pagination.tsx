import { cn } from "@/lib/utils";
import { ChevronLeft, ChevronRight } from "@unkey/icons";
import { Button } from "@unkey/ui";

export interface PaginationProps {
  page: number; // Current page (1-indexed)
  pageSize: number; // Items per page
  totalPages: number; // Total number of pages
  totalCount: number; // Total number of items
  onPageChange: (page: number) => void;
}

/**
 * Pagination component for data-table
 * Provides page navigation controls with accessibility support
 */
export function Pagination({
  page,
  pageSize,
  totalPages,
  totalCount,
  onPageChange,
}: PaginationProps) {
  const start = (page - 1) * pageSize + 1;
  const end = Math.min(page * pageSize, totalCount);

  // Generate page numbers to display
  const getPageNumbers = () => {
    const pages: (number | "ellipsis")[] = [];
    const maxVisible = 7; // Max page buttons to show

    if (totalPages <= maxVisible) {
      // Show all pages
      for (let i = 1; i <= totalPages; i++) {
        pages.push(i);
      }
    } else {
      // Always show first page
      pages.push(1);

      if (page > 3) {
        pages.push("ellipsis");
      }

      // Show pages around current page
      const startPage = Math.max(2, page - 1);
      const endPage = Math.min(totalPages - 1, page + 1);

      for (let i = startPage; i <= endPage; i++) {
        pages.push(i);
      }

      if (page < totalPages - 2) {
        pages.push("ellipsis");
      }

      // Always show last page
      pages.push(totalPages);
    }

    return pages;
  };

  const pageNumbers = getPageNumbers();

  return (
    <div className="flex items-center justify-between px-4 py-3 border-t border-gray-6">
      {/* Item range display */}
      <div className="text-sm text-gray-11">
        Showing {start}-{end} of {totalCount}
      </div>

      {/* Page navigation */}
      <div className="flex items-center gap-1">
        {/* Previous button */}
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onPageChange(page - 1)}
          disabled={page === 1}
          aria-label="Previous page"
          className="[&_svg]:size-4"
        >
          <ChevronLeft />
        </Button>

        {/* Page number buttons */}
        {pageNumbers.map((pageNum, idx) => {
          if (pageNum === "ellipsis") {
            return (
              <span key={`ellipsis-${idx === 1 ? "start" : "end"}`} className="px-2 text-gray-11">
                ...
              </span>
            );
          }

          return (
            <Button
              key={pageNum}
              variant={pageNum === page ? "outline" : "ghost"}
              size="sm"
              onClick={() => onPageChange(pageNum)}
              aria-label={`Page ${pageNum}`}
              aria-current={pageNum === page ? "page" : undefined}
              className={cn(
                "min-w-[32px]",
                pageNum === page && "bg-gray-3 text-gray-12 font-medium",
              )}
            >
              {pageNum}
            </Button>
          );
        })}

        {/* Next button */}
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onPageChange(page + 1)}
          disabled={page === totalPages}
          aria-label="Next page"
          className="[&_svg]:size-4"
        >
          <ChevronRight />
        </Button>
      </div>
    </div>
  );
}
