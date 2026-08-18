"use client";

import { formatDollars } from "@/lib/fmt";
import { cn } from "@/lib/utils";
import { CircleInfo } from "@unkey/icons";
import { InfoTooltip, ItemMedia, Skeleton } from "@unkey/ui";
import type { ReactNode } from "react";

const COLUMNS = "grid grid-cols-[1fr_8rem_11rem_6rem] items-center gap-x-4 px-4";

export function PlanTableHeader() {
  return (
    <div
      className={cn(
        COLUMNS,
        "bg-grayA-2 py-2 font-medium text-[10px] text-gray-9 uppercase tracking-wider",
      )}
    >
      <span>Product</span>
      <span>Plan</span>
      <span>Price</span>
      <span />
    </div>
  );
}

type PlanTableRowProps = {
  icon: ReactNode;
  mediaClassName: string;
  title: string;
  planName: string | null;
  feeCents: number | null;
  interval?: string;
  /**
   * Usage the fee already covers, which turns the price into "+ usage". Null for
   * products billed at a flat fee with a hard quota.
   */
  usageCreditCents?: number | null;
  action: ReactNode;
};

export function PlanTableRow({
  icon,
  mediaClassName,
  title,
  planName,
  feeCents,
  interval = "month",
  usageCreditCents = null,
  action,
}: PlanTableRowProps) {
  return (
    <div className={cn(COLUMNS, "py-3")}>
      <div className="flex items-center gap-3">
        <ItemMedia className={mediaClassName}>{icon}</ItemMedia>
        <span className="font-medium text-[13px] text-gray-12">{title}</span>
      </div>
      <span className="truncate text-[13px] text-gray-11">{planName ?? "—"}</span>
      <span className="whitespace-nowrap text-[13px] tabular-nums">
        {feeCents === null ? (
          <span className="text-gray-9">—</span>
        ) : (
          <>
            <span className="font-medium text-gray-12">{formatDollars(feeCents)}</span>
            <span className="text-gray-10">/{interval}</span>
            {usageCreditCents === null ? null : <UsageNote creditCents={usageCreditCents} />}
          </>
        )}
      </span>
      <span className="flex justify-end">{action}</span>
    </div>
  );
}

function UsageNote({ creditCents }: { creditCents: number }) {
  return (
    <InfoTooltip
      asChild
      content={
        <p className="max-w-[220px] text-[12px]">
          Includes {formatDollars(creditCents)} of usage. Extra usage is billed on top.
        </p>
      }
    >
      <span className="ml-1 inline-flex cursor-help items-center gap-1 text-gray-10">
        + usage
        <CircleInfo iconSize="sm-regular" className="text-gray-9" />
      </span>
    </InfoTooltip>
  );
}

export function PlanTableRowSkeleton({
  icon,
  mediaClassName,
}: {
  icon: ReactNode;
  mediaClassName: string;
}) {
  return (
    <div className={cn(COLUMNS, "py-3")}>
      <div className="flex items-center gap-3">
        <ItemMedia className={mediaClassName}>{icon}</ItemMedia>
        <Skeleton className="h-3.5 w-20" />
      </div>
      <Skeleton className="h-3 w-16" />
      <Skeleton className="h-3 w-24" />
      <span />
    </div>
  );
}

export function PlanTableRowMessage({ children }: { children: ReactNode }) {
  return <p className="px-4 py-3 text-[13px] text-gray-11">{children}</p>;
}
