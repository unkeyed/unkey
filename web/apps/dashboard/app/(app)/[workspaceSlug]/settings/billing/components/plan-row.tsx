"use client";

import { formatDollars } from "@/lib/fmt";
import { Button, InfoTooltip } from "@unkey/ui";
import { IconCircleInfoOutline12 } from "nucleo-ui-outline-12";
import { AdminGate } from "./admin-gate";

export function PlanTableHeader() {
  return (
    <div className="flex items-center gap-3 bg-grayA-2 px-4 py-2 font-normal text-[10px] text-gray-9 uppercase tracking-wider">
      <div className="min-w-0 flex-1">Product</div>
      <div className="w-28">Plan</div>
      <div className="w-36">Price</div>
      <div className="w-20" />
    </div>
  );
}

export function PlanName({ children }: { children: string | null }) {
  return <span className="w-28 truncate text-[13px] text-gray-11">{children ?? "—"}</span>;
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
    return <span className="w-36 text-[13px] text-gray-9">—</span>;
  }

  return (
    <span className="w-36 whitespace-nowrap text-[13px] tabular-nums">
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
            + usage
            <IconCircleInfoOutline12 className="text-gray-9" />
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
