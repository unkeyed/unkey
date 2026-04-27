"use client";

import { cn } from "@/lib/utils";
import { Fragment, type ReactNode, useEffect, useRef } from "react";
import type { Column } from "../virtual-table/types";

const SKELETON_ROWS = 16;
const ROW_HEIGHT_PX = 26;
const AUTO_SCROLL_INTERVAL_MS = 1000;
const NEAR_BOTTOM_THRESHOLD_PX = 50;

type StreamingTableProps<T> = {
  data: T[];
  columns: Column<T>[];
  keyExtractor: (item: T) => string | number;
  rowClassName?: (item: T) => string;
  onRowClick?: (item: T) => void;
  renderExpanded?: (item: T) => ReactNode;
  renderSkeletonCell: (col: Column<T>) => ReactNode;
  isLoading?: boolean;
  fixedHeight?: number;
  emptyState?: ReactNode;
};

export function StreamingTable<T>({
  data,
  columns,
  keyExtractor,
  rowClassName,
  onRowClick,
  renderExpanded,
  renderSkeletonCell,
  isLoading = false,
  fixedHeight = 500,
  emptyState,
}: StreamingTableProps<T>) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const userScrolledUp = useRef(false);

  useEffect(() => {
    const el = scrollRef.current;
    if (!el) {
      return;
    }

    const onScroll = () => {
      const distanceFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight;
      userScrolledUp.current = distanceFromBottom > NEAR_BOTTOM_THRESHOLD_PX;
    };

    el.addEventListener("scroll", onScroll, { passive: true });
    return () => el.removeEventListener("scroll", onScroll);
  }, []);

  useEffect(() => {
    if (isLoading) {
      return;
    }

    const timer = setInterval(() => {
      if (userScrolledUp.current) {
        return;
      }
      const el = scrollRef.current;
      if (el) {
        el.scrollTop = el.scrollHeight;
      }
    }, AUTO_SCROLL_INTERVAL_MS);

    return () => clearInterval(timer);
  }, [isLoading]);

  if (!isLoading && data.length === 0 && emptyState) {
    return (
      <div
        className="w-full flex justify-center items-center"
        style={{ height: `${fixedHeight}px` }}
      >
        {emptyState}
      </div>
    );
  }

  return (
    <div ref={scrollRef} className="overflow-auto" style={{ height: `${fixedHeight}px` }}>
      <table className="w-full border-separate border-spacing-0 table-auto!">
        <colgroup>
          {columns.map((col) => {
            let w = "auto";
            if (typeof col.width === "number") {
              w = `${col.width}px`;
            } else if (typeof col.width === "string") {
              w = col.width;
            }
            return <col key={col.key} style={{ width: w }} />;
          })}
        </colgroup>
        <tbody>
          {isLoading
            ? Array.from({ length: SKELETON_ROWS }).map((_, i) => (
                <tr key={`skeleton-${i}`} style={{ height: `${ROW_HEIGHT_PX}px` }}>
                  {columns.map((col, idx) => (
                    <td
                      key={col.key}
                      className={cn(
                        "text-xs align-middle whitespace-nowrap",
                        idx === 0 ? "pl-4.5" : "",
                        col.cellClassName,
                      )}
                      style={{ height: `${ROW_HEIGHT_PX}px` }}
                    >
                      {renderSkeletonCell(col)}
                    </td>
                  ))}
                </tr>
              ))
            : data.map((item) => (
                <Fragment key={keyExtractor(item)}>
                  <tr
                    onClick={onRowClick ? () => onRowClick(item) : undefined}
                    className={cn(
                      onRowClick && "cursor-pointer",
                      "transition-colors",
                      rowClassName?.(item),
                    )}
                    style={{ height: `${ROW_HEIGHT_PX}px` }}
                  >
                    {columns.map((col, idx) => (
                      <td
                        key={col.key}
                        className={cn(
                          "text-xs align-middle whitespace-nowrap pr-4",
                          idx === 0 ? "rounded-l-md" : "",
                          idx === columns.length - 1 ? "rounded-r-md" : "",
                          col.cellClassName,
                        )}
                      >
                        {col.render(item)}
                      </td>
                    ))}
                  </tr>
                  {renderExpanded?.(item)}
                </Fragment>
              ))}
        </tbody>
      </table>
    </div>
  );
}
