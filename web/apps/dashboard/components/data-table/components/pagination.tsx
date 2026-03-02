import { getPageNumbers } from "@/components/data-table/utils/get-page-numbers";
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

  const pageNumbers = getPageNumbers(page, totalPages, 7);

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
