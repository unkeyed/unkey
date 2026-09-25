"use client";

import { IconChevronExpandYOutline12 } from "@unkey/icons";
import { Skeleton } from "@unkey/ui";
import { cn } from "cn";
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
  listStatus?: ReactNode;
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
  listStatus,
  compactOnMobile = false,
}: CrumbProps) {
  return (
    <div className="flex min-w-0 items-center gap-0.5">
      <CrumbLink
        icon={icon}
        label={label}
        href={href}
        loading={loading}
        compactOnMobile={compactOnMobile}
      />
      <CrumbPopover
        items={items}
        currentId={currentId}
        searchPlaceholder={searchPlaceholder}
        emptyText={emptyText}
        footer={footer}
        listStatus={listStatus}
      >
        <button
          type="button"
          className="hidden size-6 shrink-0 items-center justify-center rounded-md text-gray-11 hover:bg-grayA-3 hover:text-gray-12 md:flex"
          aria-label={`Switch ${label}`}
        >
          <IconChevronExpandYOutline12 />
        </button>
      </CrumbPopover>
    </div>
  );
}

export function CrumbLink({
  icon,
  label,
  href,
  loading = false,
  current = false,
  compactOnMobile = false,
}: {
  icon: ReactNode;
  label: string;
  href: string;
  loading?: boolean;
  current?: boolean;
  compactOnMobile?: boolean;
}) {
  return (
    <Link
      href={href as Route}
      aria-label={label}
      aria-current={current ? "page" : undefined}
      className="flex min-w-0 items-center gap-1.5 px-1 py-1 text-[13px] font-medium text-gray-12"
    >
      {icon}
      {loading ? (
        <Skeleton className={cn("h-3 w-20 bg-gray-4", compactOnMobile && "hidden md:block")} />
      ) : (
        <span
          className={cn(
            "truncate max-w-[120px] md:max-w-[180px]",
            compactOnMobile && "hidden md:inline",
          )}
        >
          {label}
        </span>
      )}
    </Link>
  );
}

export function CrumbSeparator() {
  return <span className="select-none px-0.5 text-gray-7">/</span>;
}
