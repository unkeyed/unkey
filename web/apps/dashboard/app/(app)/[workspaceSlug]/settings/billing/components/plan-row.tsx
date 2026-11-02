"use client";

import { usePayInvoice } from "@/components/billing/payment-issues";
import { formatDollars } from "@/lib/fmt";
import type { SubscriptionProduct } from "@/lib/stripe/billingSubscriptions";
import { IconCircleInfoOutline12, IconCubeOutline18, IconNodesOutline18 } from "@unkey/icons";
import { Badge, Button, InfoTooltip, ItemContent, ItemMedia, ItemTitle, Skeleton } from "@unkey/ui";
import type { ReactNode } from "react";
import { AdminGate } from "./admin-gate";

const PRODUCT_CELL: Record<
  SubscriptionProduct,
  { title: string; icon: ReactNode; mediaClassName: string }
> = {
  api: {
    title: "API management",
    icon: <IconNodesOutline18 />,
    mediaClassName: "bg-infoA-3 text-info-11",
  },
  compute: {
    title: "Compute",
    icon: <IconCubeOutline18 />,
    mediaClassName: "bg-orangeA-3 text-orange-11",
  },
};

export function PlanProductCell({
  product,
  paymentFailed = false,
}: {
  product: SubscriptionProduct;
  paymentFailed?: boolean;
}) {
  const cell = PRODUCT_CELL[product];

  return (
    <>
      <ItemMedia className={cell.mediaClassName}>{cell.icon}</ItemMedia>
      <ItemContent>
        <ItemTitle className="flex min-w-0 items-center gap-2 overflow-hidden">
          <span className="min-w-0 truncate">{cell.title}</span>
          {paymentFailed ? (
            <Badge variant="error" size="compact" className="shrink-0">
              Payment failed
            </Badge>
          ) : null}
        </ItemTitle>
      </ItemContent>
    </>
  );
}

export function PlanTableHeader() {
  return (
    <div className="flex items-center gap-3 bg-grayA-2 px-4 py-2 font-semibold text-3xs text-gray-9 uppercase tracking-wider">
      <div className="min-w-0 flex-1">Product</div>
      <div className="w-28">Plan</div>
      <div className="w-36">Price</div>
      <div className="w-20" />
    </div>
  );
}

/** The plan, price, and action cells of a plan row while its data loads */
export function PlanCellsSkeleton() {
  return (
    <>
      <span className="flex h-5 w-28 items-center">
        <Skeleton className="h-4 w-14" />
      </span>
      <span className="flex h-5 w-36 items-center">
        <Skeleton className="h-4 w-20" />
      </span>
      <span className="flex w-20 justify-end">
        <Skeleton className="h-7 w-16 rounded-md" />
      </span>
    </>
  );
}

export function PlanName({ children }: { children: string | null }) {
  return <span className="w-28 truncate text-sm text-gray-11">{children ?? "—"}</span>;
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
    return <span className="w-36 text-sm text-gray-9">—</span>;
  }

  return (
    <span className="w-36 whitespace-nowrap text-sm tabular-nums">
      <span className="font-medium text-gray-12">{formatDollars(feeCents)}</span>
      <span className="text-gray-10">/{interval}</span>
      {usageCreditCents === null ? null : (
        <InfoTooltip
          asChild
          content={
            <p className="max-w-[220px] text-xs">
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
  product,
  paymentFailed,
  isAdmin,
  hasPlan,
  emphasize,
  onClick,
  chooseLabel,
}: {
  product: SubscriptionProduct;
  paymentFailed: boolean;
  isAdmin: boolean | undefined;
  hasPlan: boolean;
  emphasize: boolean;
  onClick: () => void;
  chooseLabel: string;
}) {
  if (paymentFailed) {
    return <PayInvoiceAction isAdmin={isAdmin} product={product} />;
  }

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
    <AdminGate isAdmin={isAdmin}>
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

function PayInvoiceAction({
  isAdmin,
  product,
}: {
  isAdmin: boolean | undefined;
  product: SubscriptionProduct;
}) {
  const payInvoice = usePayInvoice();

  return (
    <AdminGate isAdmin={isAdmin}>
      {(disabled) => (
        <Button
          variant="primary"
          size="sm"
          className="whitespace-nowrap"
          disabled={disabled || payInvoice.busy}
          loading={payInvoice.busy}
          onClick={() => payInvoice.pay(product)}
        >
          Pay invoice
        </Button>
      )}
    </AdminGate>
  );
}
