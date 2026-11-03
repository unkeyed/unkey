"use client";
import { useId, useState } from "react";
import { Button } from "../../../buttons/button";
import { Popover, PopoverContent, PopoverTrigger } from "../../../dialog/popover";
import { Input } from "../../../form/input";
import { ControlsEndPortal } from "../../../page/controls-slots";
import { Skeleton } from "../../../skeleton";
import { PaginationControls } from "./pagination-controls";

export type TablePagerProps = {
  page: number;
  onPageChange: (page: number) => void;
  loading?: boolean;
  disabled?: boolean;
} & (
  | { mode?: "pages"; pageSize: number; totalPages: number; totalCount: number }
  | { mode: "cursor"; rowCount: number; hasNextPage: boolean }
);

export function TablePager(props: TablePagerProps) {
  if (props.loading) {
    return (
      <ControlsEndPortal>
        <Skeleton className="h-7 w-40 rounded-md" />
      </ControlsEndPortal>
    );
  }
  const isEmpty =
    props.mode === "cursor" ? props.rowCount === 0 && props.page === 1 : props.totalCount === 0;
  if (isEmpty) {
    return null;
  }
  return (
    <ControlsEndPortal>
      <div className="flex shrink-0 items-center gap-1 text-sm font-medium tabular-nums text-gray-12 [&_nav_button]:text-gray-12 [&_nav_button:disabled]:text-grayA-7">
        {props.mode === "cursor" ? <CursorPager {...props} /> : <PagesPager {...props} />}
      </div>
    </ControlsEndPortal>
  );
}

function CursorPager({
  page,
  hasNextPage,
  onPageChange,
  disabled,
}: Extract<TablePagerProps, { mode: "cursor" }>) {
  return (
    <>
      <span className="whitespace-nowrap px-1">Page {page}</span>
      {page > 1 || hasNextPage ? (
        <PaginationControls
          label="Pagination navigation"
          hasPrevious={!disabled && page > 1}
          hasNext={!disabled && hasNextPage}
          onPrevious={() => onPageChange(page - 1)}
          onNext={() => onPageChange(page + 1)}
        />
      ) : null}
    </>
  );
}

function PagesPager({
  page,
  pageSize,
  totalPages,
  totalCount,
  onPageChange,
  disabled,
}: Exclude<TablePagerProps, { mode: "cursor" }>) {
  const shownPage = Math.min(page, Math.max(totalPages, 1));
  const start = (shownPage - 1) * pageSize + 1;
  const end = Math.min(shownPage * pageSize, totalCount);
  return (
    <>
      <Popover>
        <PopoverTrigger
          render={
            <Button
              variant="ghost"
              size="sm"
              className="px-1.5 text-sm font-medium text-gray-12"
              aria-label={`Rows ${start} to ${end} of ${totalCount}. Go to page`}
            />
          }
        >
          {start}–{end} of {totalCount}
        </PopoverTrigger>
        <PopoverContent align="end" className="w-auto p-3">
          <PageJump
            page={shownPage}
            totalPages={totalPages}
            onPageChange={onPageChange}
            disabled={disabled}
          />
        </PopoverContent>
      </Popover>
      {totalPages > 1 ? (
        <PaginationControls
          label="Pagination navigation"
          hasPrevious={!disabled && shownPage > 1}
          hasNext={!disabled && shownPage < totalPages}
          onPrevious={() => onPageChange(shownPage - 1)}
          onNext={() => onPageChange(shownPage + 1)}
        />
      ) : null}
    </>
  );
}

function PageJump({
  page,
  totalPages,
  onPageChange,
  disabled,
}: {
  page: number;
  totalPages: number;
  onPageChange: (page: number) => void;
  disabled?: boolean;
}) {
  const inputId = useId();
  const [draft, setDraft] = useState(String(page));
  const discard = () => setDraft(String(page));

  const commit = () => {
    if (disabled) {
      return;
    }
    const parsed = Number.parseInt(draft, 10);
    const target = Number.isFinite(parsed) ? Math.min(Math.max(1, parsed), totalPages) : page;
    setDraft(String(target));
    if (target !== page) {
      onPageChange(target);
    }
  };

  return (
    <div className="flex items-center gap-2 whitespace-nowrap text-xs text-grayA-11">
      <label htmlFor={inputId}>Go to page</label>
      <Input
        id={inputId}
        type="number"
        inputMode="numeric"
        min={1}
        max={totalPages}
        value={draft}
        readOnly={disabled}
        aria-disabled={disabled || undefined}
        onChange={(event) => setDraft(event.target.value)}
        onBlur={discard}
        onKeyDown={(event) => {
          if (event.key === "Enter") {
            event.preventDefault();
            commit();
          } else if (event.key === "Escape") {
            discard();
          }
        }}
        className="h-7 w-14 text-center tabular-nums [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none"
      />
      <span className="tabular-nums">of {totalPages}</span>
    </div>
  );
}
