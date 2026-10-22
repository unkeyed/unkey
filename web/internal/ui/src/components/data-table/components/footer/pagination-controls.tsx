"use client";
import { IconChevronLeftOutline18, IconChevronRightOutline18 } from "@unkey/icons";
import type { ReactNode } from "react";
import { cn } from "../../../../lib/utils";
import { Button } from "../../../buttons/button";

export type PaginationControlsProps = {
  label: string;
  hasPrevious: boolean;
  hasNext: boolean;
  onPrevious: () => void;
  onNext: () => void;
  className?: string;
  children?: ReactNode;
};

export function PaginationControls({
  label,
  hasPrevious,
  hasNext,
  onPrevious,
  onNext,
  className,
  children,
}: PaginationControlsProps) {
  return (
    <nav aria-label={label} className={cn("flex items-center justify-center gap-1", className)}>
      <Button
        variant="ghost"
        size="icon"
        onClick={onPrevious}
        disabled={!hasPrevious}
        aria-label="Go to previous page"
        className="border-none text-grayA-9 hover:text-grayA-11 disabled:pointer-events-none disabled:opacity-80"
      >
        <IconChevronLeftOutline18 />
      </Button>
      {children}
      <Button
        variant="ghost"
        size="icon"
        onClick={onNext}
        disabled={!hasNext}
        aria-label="Go to next page"
        className="border-none text-grayA-9 hover:text-grayA-11 disabled:pointer-events-none disabled:opacity-80"
      >
        <IconChevronRightOutline18 />
      </Button>
    </nav>
  );
}
