import { cn } from "@/lib/utils";
import { ArrowsToAllDirections, ArrowsToCenter, ChevronLeft, ChevronRight } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useCallback, useState } from "react";

interface PaginationFooterProps {
  page: number;
  pageSize: number;
  totalPages: number;
  totalCount: number;
  onPageChange: (page: number) => void;
  itemLabel?: string;
  hide?: boolean;
  headerContent?: React.ReactNode;
}

export function PaginationFooter({
  page,
  pageSize,
  totalPages,
  totalCount,
  onPageChange,
  itemLabel = "items",
  hide,
  headerContent,
}: PaginationFooterProps) {
  const [isOpen, setIsOpen] = useState(true);

  const handleClose = useCallback(() => {
    setIsOpen(false);
  }, []);

  const handleOpen = useCallback(() => {
    setIsOpen(true);
  }, []);

  const start = (page - 1) * pageSize + 1;
  const end = Math.min(page * pageSize, totalCount);

  const getPageNumbers = (): Array<number | "ellipsis"> => {
    const pages: Array<number | "ellipsis"> = [];
    const maxVisible = 5;

    if (totalPages <= maxVisible) {
      for (let i = 1; i <= totalPages; i++) {
        pages.push(i);
      }
    } else {
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
    }

    return pages;
  };

  if (hide) {
    return null;
  }

  // Minimized state - parked at right side
  if (!isOpen) {
    return (
      <div className="fixed bottom-6 right-6 z-10 transition-all duration-300 ease-out animate-slide-in-from-bottom">
        <button
          type="button"
          onClick={handleOpen}
          className="bg-gray-1 dark:bg-black border border-gray-6 rounded-lg shadow-lg p-3 transition-all duration-200 hover:shadow-xl hover:scale-105 group"
          title={`Page ${page} of ${totalPages} • ${start}-${end} of ${totalCount} ${itemLabel}`}
        >
          <div className="flex items-center gap-2">
            <span className="text-[11px] text-gray-9 font-medium">
              {start}-{end} of {totalCount}
            </span>
            <div className="w-px h-3 bg-gray-6" />
            <span className="text-[12px] font-medium text-gray-11 group-hover:text-gray-12 transition-colors">
              Page {page}/{totalPages}
            </span>
            <ArrowsToAllDirections iconSize="sm-regular" />
          </div>
        </button>
      </div>
    );
  }

  const pageNumbers = getPageNumbers();

  return (
    <div
      className={cn(
        "fixed bottom-0 left-0 right-0 w-full items-center justify-center flex z-10 transition-all duration-300 ease-out pointer-events-none",
        "opacity-100 animate-slide-up-from-bottom",
      )}
    >
      <div className="w-[740px] border bg-gray-1 dark:bg-black border-gray-6 min-h-[60px] flex items-center justify-center rounded-[10px] drop-shadow-lg transform-gpu shadow-sm mb-5 transition-all duration-200 hover:shadow-lg pointer-events-auto">
        <div className="flex flex-col w-full">
          {/* Header content */}
          {headerContent && (
            <div
              className="transition-all duration-200 animate-fade-in-up"
              style={{ animationDelay: "0.2s" }}
            >
              {headerContent}
            </div>
          )}

          <div
            className="flex w-full justify-between items-center text-[13px] text-accent-9 p-[18px] transition-all duration-200 animate-fade-in-up"
            style={{ animationDelay: "0.3s" }}
          >
            {/* Item count */}
            <div className="flex gap-2">
              <span>Viewing</span>
              <span className="text-accent-12 transition-colors duration-200">
                {start}-{end}
              </span>
              <span>of</span>
              <span className="text-grayA-12 transition-colors duration-200">{totalCount}</span>
              <span>{itemLabel}</span>
            </div>

            {/* Pagination controls */}
            <div className="items-center flex gap-2">
              {/* Previous button */}
              <Button
                variant="ghost"
                size="icon"
                onClick={() => onPageChange(page - 1)}
                disabled={page === 1}
                aria-label="Previous page"
                className="border-none disabled:pointer-events-none"
              >
                <ChevronLeft iconSize="sm-regular" />
              </Button>

              {/* Page numbers */}
              <div className="flex items-center gap-1">
                {pageNumbers.map((pageNum, idx) => {
                  if (pageNum === "ellipsis") {
                    return (
                      <span
                        key={idx < pageNumbers.length / 2 ? "ellipsis-start" : "ellipsis-end"}
                        className="px-2 text-gray-9"
                      >
                        ...
                      </span>
                    );
                  }

                  const isCurrentPage = pageNum === page;
                  return (
                    <Button
                      key={pageNum}
                      variant={isCurrentPage ? "outline" : "ghost"}
                      size="sm"
                      onClick={() => onPageChange(pageNum)}
                      aria-label={`Page ${pageNum}`}
                      aria-current={isCurrentPage ? "page" : undefined}
                      className={cn(
                        "min-w-[32px] transition-all",
                        isCurrentPage && "bg-gray-3 text-gray-12 font-medium pointer-events-none",
                      )}
                    >
                      {pageNum}
                    </Button>
                  );
                })}
              </div>

              {/* Next button */}
              <Button
                variant="ghost"
                size="icon"
                onClick={() => onPageChange(page + 1)}
                disabled={page === totalPages}
                aria-label="Next page"
                className="border-none disabled:pointer-events-none"
              >
                <ChevronRight iconSize="sm-regular" />
              </Button>

              {/* Minimize button */}
              <div
                className="flex justify-end transition-all duration-200 animate-fade-in-down"
                style={{ animationDelay: "0.1s" }}
              >
                <Button
                  size="icon"
                  variant="ghost"
                  className="[&_svg]:size-[14px] transition-all duration-200 rounded hover:bg-gray-3 transform hover:scale-110"
                  onClick={handleClose}
                  aria-label="Minimize"
                  title="Minimize"
                >
                  <ArrowsToCenter iconSize="lg-regular" />
                </Button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
