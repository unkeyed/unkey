"use client";

import { cn } from "@/lib/utils";
import { ChevronExpandY } from "@unkey/icons";
import type { Route } from "next";
import Link from "next/link";
import type { ReactNode } from "react";
import { CrumbPopover, type CrumbPopoverFooter, type CrumbPopoverItem } from "./crumb-popover";

type CrumbProps = {
  icon: ReactNode;
  label: string;
  href: string;
  items: CrumbPopoverItem[];
  currentId: string;
  searchPlaceholder: string;
  emptyText: string;
  footer: CrumbPopoverFooter;
  loading?: boolean;
  compactOnMobile?: boolean;
};

export function Crumb({
  icon,
  label,
  href,
  items,
  currentId,
  searchPlaceholder,
  emptyText,
  footer,
  loading = false,
  compactOnMobile = false,
}: CrumbProps) {
  return (
    <div className="flex min-w-0 items-center gap-0.5">
      <Link
        href={href as Route}
        aria-label={label}
        className="flex min-w-0 items-center gap-1.5 px-1 py-1 text-[13px] font-medium text-accent-12"
      >
        {icon}
        {loading ? (
          <span
            aria-hidden="true"
            className={cn(
              "h-3 w-20 rounded-sm bg-gray-4 animate-pulse",
              compactOnMobile && "hidden md:block",
            )}
          />
        ) : (
          <span
            className={cn(
              "max-w-[120px] truncate md:max-w-[180px]",
              compactOnMobile && "hidden md:inline",
            )}
          >
            {label}
          </span>
        )}
      </Link>
      <CrumbPopover
        items={items}
        currentId={currentId}
        searchPlaceholder={searchPlaceholder}
        emptyText={emptyText}
        footer={footer}
      >
        <button
          type="button"
          className="hidden size-6 shrink-0 items-center justify-center rounded-md text-gray-11 hover:bg-grayA-3 hover:text-accent-12 md:flex"
          aria-label={`Switch ${label}`}
        >
          <ChevronExpandY className="size-3" iconSize="sm-regular" />
        </button>
      </CrumbPopover>
    </div>
  );
}

export function CrumbSeparator() {
  return <span className="select-none px-0.5 text-gray-7">/</span>;
}
