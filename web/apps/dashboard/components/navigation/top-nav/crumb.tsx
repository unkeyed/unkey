"use client";

import { ChevronExpandY } from "@unkey/icons";
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
}: CrumbProps) {
  return (
    <div className="flex min-w-0 items-center gap-0.5">
      <Link
        href={href}
        className="flex min-w-0 items-center gap-1.5 px-1 py-1 text-[13px] font-medium text-accent-12"
      >
        {icon}
        <span className="truncate max-w-[120px] md:max-w-[180px]">{label}</span>
      </Link>
      {/* Sibling switcher is desktop-only. On mobile the crumb is a
          pure link; lateral navigation uses the section list pages. */}
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
