import type { Table } from "@tanstack/react-table";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "~/components/ui/button";

const MANY_THRESHOLD = 100;

type Props<T> = {
  table: Table<T>;
};

export function KeysPagination<T>({ table }: Props<T>) {
  const pageCount = table.getPageCount();
  if (pageCount <= 1) return null;

  const pageIndex = table.getState().pagination.pageIndex;
  const totalLabel = pageCount > MANY_THRESHOLD ? "Many" : pageCount.toLocaleString();

  return (
    <div className="flex items-center gap-1 text-gray-11 text-xs">
      <span className="px-1">
        Page <span className="font-medium text-gray-12">{(pageIndex + 1).toLocaleString()}</span>{" "}
        of <span className="font-medium text-gray-12">{totalLabel}</span>
      </span>
      <Button
        size="icon"
        variant="ghost"
        className="size-6"
        onClick={() => table.previousPage()}
        disabled={!table.getCanPreviousPage()}
        aria-label="Previous page"
      >
        <ChevronLeft />
      </Button>
      <Button
        size="icon"
        variant="ghost"
        className="size-6"
        onClick={() => table.nextPage()}
        disabled={!table.getCanNextPage()}
        aria-label="Next page"
      >
        <ChevronRight />
      </Button>
    </div>
  );
}
