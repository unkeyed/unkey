"use client";
import { ArrowsToAllDirections, ArrowsToCenter, ChevronLeft, ChevronRight } from "@unkey/icons";
import { memo, useMemo, useState } from "react";
import { cn } from "../../../../lib/utils";
import { Button } from "../../../buttons/button";
import { getPageNumbers } from "../../utils/get-page-numbers";
import { PaginationFooterSkeleton } from "../skeletons/pagination-footer-skeleton";

export interface PaginationFooterProps {
  page: number;
  pageSize: number;
  totalPages: number;
  totalCount: number;
  onPageChange: (page: number) => void;
  itemLabel?: string;
  hide?: boolean;
  loading?: boolean;
  disabled?: boolean;
  headerContent?: React.ReactNode;
}

export const PaginationFooter = memo(function PaginationFooter({
  page,
  pageSize,
  totalPages,
  totalCount,
  onPageChange,
  itemLabel = "items",
  hide,
  loading,
  disabled,
  headerContent,
}: PaginationFooterProps) {
  const [isOpen, setIsOpen] = useState(true);

  const start = (page - 1) * pageSize + 1;
  const end = Math.min(page * pageSize, totalCount);
  const pageNumbers = useMemo(() => getPageNumbers(page, totalPages), [page, totalPages]);

  if (hide) {
    return null;
  }

  // Minimized state - parked at right side
  if (!isOpen) {
    return (
      <div className="fixed bottom-6 right-6 z-10 animate-fade-slide-in">
        <button
          type="button"
          onClick={() => setIsOpen(true)}
          className="bg-gray-1 dark:bg-black border border-gray-6 rounded-lg shadow-lg p-3 duration-200 hover:shadow-xl hover:scale-105 group"
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

  return (
    <div
      className={cn(
        "fixed bottom-0 left-0 right-0 w-full items-center justify-center flex z-10 animation-ease-out pointer-events-none",
        "opacity-100",
      )}
    >
      {loading ? (
        <PaginationFooterSkeleton />
      ) : (
        <div className="w-[740px] border bg-gray-1 dark:bg-black border-gray-6 min-h-[60px] flex items-center justify-center rounded-[10px] drop-shadow-lg transform-gpu shadow-sm mb-5 transition-all duration-200 hover:shadow-lg pointer-events-auto">
          <div className="flex flex-col w-full animate-ease-in">
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
              <nav aria-label="Pagination navigation" className="flex items-center gap-1">
                {/* Previous button */}
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => onPageChange(page - 1)}
                  disabled={disabled || page === 1}
                  aria-label="Go to previous page"
                  className="border-none disabled:pointer-events-none disabled:opacity-30 focus:ring-0"
                >
                  <ChevronLeft iconSize="sm-regular" />
                </Button>

                {/* Page number segmented group */}
                <div
                  aria-label="Page numbers"
                  className="flex items-center justify-center bg-grayA-2 border border-grayA-3 rounded-lg p-0.5 gap-0.5 transition-all animate-fade-in duration-500"
                >
                  {pageNumbers.map((pageNum, idx) => {
                    if (pageNum === "ellipsis") {
                      return (
                        <span
                          key={idx < pageNumbers.length / 2 ? "ellipsis-start" : "ellipsis-end"}
                          aria-hidden="true"
                          className="w-7 h-7 flex items-center justify-center text-gray-11 text-[10px] tracking-widest select-none"
                        >
                          ···
                        </span>
                      );
                    }

                    const isCurrentPage = pageNum === page;
                    return (
                      <button
                        key={pageNum}
                        type="button"
                        onClick={() => {
                          if (!isCurrentPage && !disabled) {
                            onPageChange(pageNum);
                          }
                        }}
                        disabled={disabled && !isCurrentPage}
                        aria-label={`Page ${pageNum}`}
                        aria-current={isCurrentPage ? "page" : undefined}
                        className={cn(
                          "w-7 h-7 flex items-center justify-center rounded-md text-xs font-medium cursor-pointer",
                          isCurrentPage
                            ? "bg-grayA-5 text-accent-12 shadow-sm pointer-events-none ring-0 border border-grayA-3 scale-105 text-sm transition-all duration-300"
                            : "text-gray-11 hover:text-gray-12 hover:bg-grayA-3",
                          disabled && !isCurrentPage && "opacity-30 pointer-events-none",
                        )}
                      >
                        {pageNum}
                      </button>
                    );
                  })}
                </div>

                {/* Next button */}
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => onPageChange(page + 1)}
                  disabled={disabled || page === totalPages}
                  aria-label="Go to next page"
                  className="border-none disabled:pointer-events-none disabled:opacity-30 focus:ring-0"
                >
                  <ChevronRight iconSize="sm-regular" />
                </Button>

                {/* Minimize button */}
                <div
                  className="flex justify-end transition-all duration-200 animate-fade-in-down ml-1"
                  style={{ animationDelay: "0.1s" }}
                >
                  <Button
                    size="icon"
                    variant="ghost"
                    className="[&_svg]:size-[14px] transition-all duration-200 rounded hover:bg-gray-3 transform hover:scale-110"
                    onClick={() => setIsOpen(false)}
                    aria-label="Minimize"
                    title="Minimize"
                  >
                    <ArrowsToCenter iconSize="lg-regular" />
                  </Button>
                </div>
              </nav>
            </div>
          </div>
        </div>
      )}
    </div>
  );
});
