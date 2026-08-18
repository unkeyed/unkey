"use client";

import { formatDollars } from "@/lib/fmt";
import { cn } from "@/lib/utils";
import { CircleInfo } from "@unkey/icons";
import { Button, InfoTooltip, Item, ItemMedia, ItemTitle, Skeleton } from "@unkey/ui";
import type { ReactNode } from "react";
import { AdminGate } from "./admin-gate";

/**
 * The header and every row are separate grids, so the last three columns are
 * fixed widths to line up across them. Only product flexes, and it truncates.
 */
const COLUMNS =
  "grid grid-cols-[minmax(0,1fr)_5.5rem_6.5rem_5.5rem] items-center gap-x-3 lg:grid-cols-[minmax(0,1fr)_7rem_9.5rem_5.5rem] lg:gap-x-4";

export function PlanTableHeader() {
  return (
    <div
      className={cn(
        COLUMNS,
        "bg-grayA-2 px-4 py-2 font-semibold text-[10px] text-gray-9 uppercase tracking-wider",
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
  /** Null for products billed at a flat fee with a hard quota. */
  usageCreditCents?: number | null;
  /** A mid-cycle plan change prorates the credit, so this period gets less than the fee. */
  usageCreditProrated?: boolean;
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
  usageCreditProrated = false,
  action,
}: PlanTableRowProps) {
  return (
    <Item className={COLUMNS}>
      <div className="flex min-w-0 items-center gap-3">
        <ItemMedia className={mediaClassName}>{icon}</ItemMedia>
        <ItemTitle className="truncate">{title}</ItemTitle>
      </div>
      <span className="truncate text-[13px] text-gray-11">{planName ?? "—"}</span>
      <span className="whitespace-nowrap text-[13px] tabular-nums">
        {feeCents === null ? (
          <span className="text-gray-9">—</span>
        ) : (
          <>
            <span className="font-medium text-gray-12">{formatDollars(feeCents)}</span>
            <span className="text-gray-10">/{interval}</span>
            {usageCreditCents === null ? null : (
              <UsageNote creditCents={usageCreditCents} prorated={usageCreditProrated} />
            )}
          </>
        )}
      </span>
      <span className="flex justify-end">{action}</span>
    </Item>
  );
}

function UsageNote({ creditCents, prorated }: { creditCents: number; prorated: boolean }) {
  return (
    <InfoTooltip
      asChild
      content={
        <p className="max-w-[220px] text-[12px]">
          Includes {formatDollars(creditCents)} of usage
          {prorated ? " this period, prorated" : ""}. Extra usage is billed on top.
        </p>
      }
    >
      <span className="ml-1 inline-flex cursor-help items-center gap-1 text-gray-10">
        <span className="hidden lg:inline">+ usage</span>
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
    <Item className={COLUMNS}>
      <div className="flex min-w-0 items-center gap-3">
        <ItemMedia className={mediaClassName}>{icon}</ItemMedia>
        <Skeleton className="h-3.5 w-20" />
      </div>
      <Skeleton className="h-3 w-16" />
      <Skeleton className="h-3 w-24" />
      <span />
    </Item>
  );
}

export function PlanTableRowMessage({
  icon,
  mediaClassName,
  title,
  children,
}: {
  icon: ReactNode;
  mediaClassName: string;
  title: string;
  children: ReactNode;
}) {
  return (
    <Item className={COLUMNS}>
      <div className="flex min-w-0 items-center gap-3">
        <ItemMedia className={mediaClassName}>{icon}</ItemMedia>
        <ItemTitle className="truncate">{title}</ItemTitle>
      </div>
      <p className="col-span-3 text-[13px] text-gray-11">{children}</p>
    </Item>
  );
}

export function PlanRowAction({
  isAdmin,
  hasPlan,
  hasPaymentMethod,
  needsPaymentReason,
  emphasize,
  onClick,
  chooseLabel,
}: {
  isAdmin: boolean | undefined;
  hasPlan: boolean;
  hasPaymentMethod: boolean;
  needsPaymentReason: string;
  emphasize: boolean;
  onClick: () => void;
  chooseLabel: string;
}) {
  if (hasPlan) {
    return (
      <AdminGate isAdmin={isAdmin}>
        {(disabled) => (
          <Button variant="outline" size="sm" disabled={disabled} onClick={onClick}>
            Change
          </Button>
        )}
      </AdminGate>
    );
  }

  return (
    <AdminGate isAdmin={isAdmin} blocked={!hasPaymentMethod} blockedReason={needsPaymentReason}>
      {(disabled) => (
        <Button
          variant={emphasize ? "primary" : "outline"}
          size="sm"
          disabled={disabled}
          onClick={onClick}
        >
          {chooseLabel}
        </Button>
      )}
    </AdminGate>
  );
}
