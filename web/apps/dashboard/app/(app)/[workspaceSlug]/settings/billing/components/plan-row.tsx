"use client";

import { formatDollars } from "@/lib/fmt";
import { cn } from "@/lib/utils";
import { CircleInfo } from "@unkey/icons";
import { Button, InfoTooltip } from "@unkey/ui";
import { AdminGate } from "./admin-gate";

/**
 * The header and every row are separate grids, so the last three columns are
 * fixed widths to line up across them. Only product flexes, and it truncates.
 */
export const PLAN_COLUMNS =
  "grid grid-cols-[minmax(0,1fr)_5.5rem_6.5rem_5.5rem] items-center gap-x-3 lg:grid-cols-[minmax(0,1fr)_7rem_9.5rem_5.5rem] lg:gap-x-4";

export function PlanTableHeader() {
  return (
    <div
      className={cn(
        PLAN_COLUMNS,
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

export function PlanName({ children }: { children: string | null }) {
  return <span className="truncate text-[13px] text-gray-11">{children ?? "—"}</span>;
}

export function PlanPrice({
  feeCents,
  interval = "month",
  usageCreditCents = null,
  usageCreditProrated = false,
}: {
  feeCents: number | null;
  interval?: string;
  /** Null for products billed at a flat fee with a hard quota. */
  usageCreditCents?: number | null;
  /** A mid-cycle plan change prorates the credit, so this period gets less than the fee. */
  usageCreditProrated?: boolean;
}) {
  if (feeCents === null) {
    return <span className="text-[13px] text-gray-9">—</span>;
  }

  return (
    <span className="whitespace-nowrap text-[13px] tabular-nums">
      <span className="font-medium text-gray-12">{formatDollars(feeCents)}</span>
      <span className="text-gray-10">/{interval}</span>
      {usageCreditCents === null ? null : (
        <InfoTooltip
          asChild
          content={
            <p className="max-w-[220px] text-[12px]">
              Includes {formatDollars(usageCreditCents)} of usage
              {usageCreditProrated ? " this period, prorated" : ""}. Extra usage is billed on top.
            </p>
          }
        >
          <span className="ml-1 inline-flex cursor-help items-center gap-1 text-gray-10">
            <span className="hidden lg:inline">+ usage</span>
            <CircleInfo iconSize="sm-regular" className="text-gray-9" />
          </span>
        </InfoTooltip>
      )}
    </span>
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
