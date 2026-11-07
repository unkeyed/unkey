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
  tone?: PaginationTone;
  children?: ReactNode;
};

type PaginationTone = "muted" | "strong";

const TONE_CLASS: Record<PaginationTone, string> = {
  muted: "text-grayA-9 hover:text-grayA-11",
  strong: "text-gray-12",
};

export function PaginationControls({
  label,
  hasPrevious,
  hasNext,
  onPrevious,
  onNext,
  className,
  tone = "muted",
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
        className={cn(
          "border-none disabled-state:pointer-events-none disabled-state:opacity-80",
          TONE_CLASS[tone],
        )}
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
        className={cn(
          "border-none disabled-state:pointer-events-none disabled-state:opacity-80",
          TONE_CLASS[tone],
        )}
      >
        <IconChevronRightOutline18 />
      </Button>
    </nav>
  );
}
