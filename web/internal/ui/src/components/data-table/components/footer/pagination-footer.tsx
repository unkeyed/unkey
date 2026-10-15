"use client";
import {
  IconChevronLeftOutline18,
  IconChevronRightOutline18,
  IconMaximizeWindowOutline18,
  IconMinimizeWindowOutline18,
} from "@unkey/icons";
import { memo, useMemo, useState } from "react";
import { cn } from "../../../../lib/utils";
import { Button } from "../../../buttons/button";
import { FOOTER_PANEL } from "../../constants/constants";
import { getPageNumbers } from "../../utils/get-page-numbers";
import { PaginationFooterSkeleton } from "../skeletons/pagination-footer-skeleton";

type PaginationFooterCommonProps = {
  page: number;
  onPageChange: (page: number) => void;
  itemLabel?: string;
  hide?: boolean;
  loading?: boolean;
  disabled?: boolean;
  headerContent?: React.ReactNode;
};

export type PaginationFooterProps = PaginationFooterCommonProps &
  (
    | { mode?: "pages"; pageSize: number; totalPages: number; totalCount: number }
    | { mode: "cursor"; rowCount: number; hasNextPage: boolean }
  );

export const PaginationFooter = memo(function PaginationFooter(props: PaginationFooterProps) {
  const { page, onPageChange, itemLabel = "items", hide, loading, disabled, headerContent } = props;
  const [isOpen, setIsOpen] = useState(true);

  const start = props.mode === "cursor" ? 0 : (page - 1) * props.pageSize + 1;
  const end =
    props.mode === "cursor" ? props.rowCount : Math.min(page * props.pageSize, props.totalCount);
  const totalPages = props.mode === "cursor" ? page : props.totalPages;
  const pageNumbers = useMemo(() => getPageNumbers(page, totalPages), [page, totalPages]);
  const showNavigation = props.mode === "cursor" ? page > 1 || props.hasNextPage : totalPages > 1;
  const hasNextPage = props.mode === "cursor" ? props.hasNextPage : page < totalPages;
  const rangeLabel =
    props.mode === "cursor"
      ? `${props.rowCount} ${itemLabel}`
      : `${start}-${end} of ${props.totalCount}`;
  const pageLabel = props.mode === "cursor" ? `Page ${page}` : `Page ${page}/${totalPages}`;
  const isEmpty =
    props.mode === "cursor" ? props.rowCount === 0 && page === 1 : props.totalCount === 0;

  if (hide || (!loading && isEmpty)) {
    return null;
  }

  // Minimized state - parked at right side
  if (!isOpen) {
    return (
      <div className="fixed bottom-6 right-6 z-10 animate-fade-slide-in">
        <button
          type="button"
          onClick={() => setIsOpen(true)}
          className="cursor-pointer text-grayA-9 hover:text-grayA-11 bg-raised rounded-lg shadow-floating p-3 duration-200 hover:scale-105 group"
          title={
            props.mode === "cursor"
              ? `${pageLabel} • ${rangeLabel}`
              : `Page ${page} of ${totalPages} • ${rangeLabel} ${itemLabel}`
          }
        >
          <div className="flex items-center gap-2">
            <span className="text-2xs font-medium">{rangeLabel}</span>
            {showNavigation ? (
              <>
                <div className="w-px h-3 bg-gray-6" />
                <span className="text-xs font-medium mr-2">{pageLabel}</span>
              </>
            ) : null}
            <IconMaximizeWindowOutline18 className="size-4" />
          </div>
        </button>
      </div>
    );
  }

  return (
    <div
      className={cn(
        "fixed bottom-0 left-0 right-0 w-full items-center justify-center flex flex-col z-10 pointer-events-none",
        "opacity-100",
      )}
    >
      {loading ? (
        <PaginationFooterSkeleton />
      ) : (
        <div className={cn(FOOTER_PANEL, "transition-all duration-200 pointer-events-auto")}>
          <div className="flex flex-col w-full">
            {/* Header content */}
            {headerContent && <div className="flex items-center w-full">{headerContent}</div>}
            <div className="flex w-full justify-between items-center text-sm text-grayA-9 p-[18px] min-h-[60px]">
              {/* Item count */}
              <div className="flex gap-2">
                <span>Viewing</span>
                <span className="text-grayA-11 transition-colors duration-200">
                  {props.mode === "cursor" ? props.rowCount : `${start}-${end}`}
                </span>
                {props.mode !== "cursor" && (
                  <>
                    <span>of</span>
                    <span className="text-grayA-11 transition-colors duration-200">
                      {props.totalCount}
                    </span>
                  </>
                )}
                <span>{itemLabel}</span>
              </div>

              {/* Pagination controls */}
              <nav
                aria-label="Pagination navigation"
                className="flex items-center justify-center gap-1"
              >
                {/* Previous button */}
                {showNavigation ? (
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => onPageChange(page - 1)}
                    disabled={disabled || page === 1}
                    aria-label="Go to previous page"
                    className="border-none text-grayA-9 hover:text-grayA-11 disabled:pointer-events-none disabled:opacity-80 focus:ring-0"
                  >
                    <IconChevronLeftOutline18 />
                  </Button>
                ) : null}
                {/* Page number segmented group */}
                {showNavigation ? (
                  props.mode === "cursor" ? (
                    <span aria-current="page" className="px-2 text-xs font-medium">
                      {pageLabel}
                    </span>
                  ) : (
                    <div
                      aria-label="Page numbers"
                      className="flex items-center justify-center rounded-lg p-0.5 gap-0.5 transition-all animate-fade-in duration-500"
                    >
                      {pageNumbers.map((pageNum, idx) => {
                        if (pageNum === "ellipsis") {
                          return (
                            <span
                              key={idx < pageNumbers.length / 2 ? "ellipsis-start" : "ellipsis-end"}
                              aria-hidden="true"
                              className="w-7 h-7 flex items-center justify-center text-grayA-9 text-3xs tracking-widest select-none"
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
                                ? "text-gray-12 pointer-events-none ring-0 border bg-grayA-4 text-sm transition-all duration-300"
                                : "text-grayA-9 hover:text-gray-12 hover:bg-grayA-3",
                              disabled && !isCurrentPage && "opacity-80 pointer-events-none",
                            )}
                          >
                            {pageNum}
                          </button>
                        );
                      })}
                    </div>
                  )
                ) : null}
                {/* Next button */}
                {showNavigation ? (
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => onPageChange(page + 1)}
                    disabled={disabled || !hasNextPage}
                    aria-label="Go to next page"
                    className="border-none text-grayA-9 hover:text-grayA-11 disabled:pointer-events-none disabled:opacity-80 focus:ring-0 mr-2"
                  >
                    <IconChevronRightOutline18 />
                  </Button>
                ) : null}
              </nav>
              {/* Minimize button */}
              <div
                className="flex justify-end transition-all duration-200 animate-fade-in-down ml-4 rounded"
                style={{ animationDelay: "0.1s" }}
              >
                <Button
                  size="icon"
                  variant="ghost"
                  className="[&_svg]:size-6 text-grayA-9 hover:text-grayA-11 transition-all duration-200 rounded transform hover:scale-110"
                  onClick={() => setIsOpen(false)}
                  aria-label="Minimize"
                  title="Minimize"
                >
                  <IconMinimizeWindowOutline18 />
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
});
