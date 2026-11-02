"use client";

import { usePayInvoice, usePaymentIssues } from "@/hooks/use-payment-issues";
import { formatDollars } from "@/lib/fmt";
import type { SubscriptionProduct } from "@/lib/stripe/billingSubscriptions";
import { IconCircleInfoOutline12, IconCubeOutline18, IconNodesOutline18 } from "@unkey/icons";
import {
  Badge,
  Button,
  InfoTooltip,
  Item,
  ItemActions,
  ItemContent,
  ItemMedia,
  ItemSeparator,
  ItemTitle,
  Skeleton,
} from "@unkey/ui";
import { cn } from "cn";
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

function PlanProductCell({
  product,
  paymentFailed = false,
}: {
  product: SubscriptionProduct;
  paymentFailed?: boolean;
}) {
  const cell = PRODUCT_CELL[product];

  return (
    <>
      <ItemMedia className={cn("hidden @4xl:flex", cell.mediaClassName)}>{cell.icon}</ItemMedia>
      <ItemContent>
        <ItemTitle className="flex min-w-0 items-center gap-2 overflow-hidden @xl:flex-col @xl:items-start @xl:gap-1 @4xl:flex-row @4xl:items-center @4xl:gap-2">
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

function PlanRow({ children }: { children: ReactNode }) {
  return <Item className="flex-wrap gap-y-2 @xl:flex-nowrap">{children}</Item>;
}

export function PlanTable({ children }: { children: ReactNode }) {
  const hasPaymentIssue = usePaymentIssues().length > 0;

  return (
    <div
      className={cn(
        "@container flex flex-col",
        hasPaymentIssue ? "[--plan-action-w:--spacing(44)]" : "[--plan-action-w:--spacing(20)]",
      )}
    >
      <PlanTableHeader />
      {children}
    </div>
  );
}

export function PlanRowLayout({
  product,
  paymentFailed = false,
  name,
  price,
  action,
}: {
  product: SubscriptionProduct;
  paymentFailed?: boolean;
  name: ReactNode;
  price: ReactNode;
  action: ReactNode;
}) {
  return (
    <PlanRow>
      <PlanProductCell product={product} paymentFailed={paymentFailed} />
      <ItemActions className="w-full flex-wrap gap-x-3 gap-y-2 @xl:w-auto @xl:flex-nowrap">
        {name}
        {price}
        <span className="flex w-full justify-start @xl:w-(--plan-action-w) @xl:justify-end">
          {action}
        </span>
      </ItemActions>
    </PlanRow>
  );
}

export function PlanRowMessage({
  product,
  children,
}: {
  product: SubscriptionProduct;
  children: ReactNode;
}) {
  return (
    <PlanRow>
      <PlanProductCell product={product} />
      <p className="basis-full text-sm text-gray-11 @xl:basis-auto">{children}</p>
    </PlanRow>
  );
}

export function PlanRowSkeleton({ product }: { product: SubscriptionProduct }) {
  return (
    <PlanRowLayout
      product={product}
      name={
        <span className="flex h-5 items-center @xl:w-24 @4xl:w-28">
          <Skeleton className="h-4 w-14" />
        </span>
      }
      price={
        <span className="flex h-5 items-center @xl:w-24 @4xl:w-36">
          <Skeleton className="h-4 w-20" />
        </span>
      }
      action={<Skeleton className="h-7 w-16 rounded-md" />}
    />
  );
}

function PlanTableHeader() {
  return (
    <>
      <ItemSeparator />
      <div className="hidden items-center gap-3 bg-grayA-2 px-4 py-2 font-semibold text-3xs text-gray-9 uppercase tracking-wider @xl:flex">
        <div className="min-w-0 flex-1">Product</div>
        <div className="w-24 @4xl:w-28">Plan</div>
        <div className="w-24 @4xl:w-36">Price</div>
        <div className="w-(--plan-action-w)" />
      </div>
      <ItemSeparator className="hidden @xl:block" />
    </>
  );
}

export function PlanName({ children }: { children: string | null }) {
  return (
    <span className="min-w-0 truncate text-sm text-gray-11 @xl:w-24 @4xl:w-28">
      {children ?? "—"}
    </span>
  );
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
    return <span className="text-sm text-gray-9 @xl:w-24 @4xl:w-36">—</span>;
  }

  return (
    <span className="whitespace-nowrap text-sm tabular-nums @xl:whitespace-normal @4xl:whitespace-nowrap @xl:w-24 @4xl:w-36">
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
          <span className="ml-1 inline-flex cursor-help items-center gap-1 text-gray-10 @xl:ml-0 @4xl:ml-1">
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
  const planAction = (
    <PlanChangeAction
      isAdmin={isAdmin}
      hasPlan={hasPlan}
      emphasize={emphasize && !paymentFailed}
      onClick={onClick}
      chooseLabel={chooseLabel}
    />
  );

  if (!paymentFailed) {
    return planAction;
  }

  return (
    <span className="flex flex-wrap justify-start gap-2 @xl:flex-nowrap @xl:justify-end">
      <PayInvoiceAction isAdmin={isAdmin} product={product} />
      {planAction}
    </span>
  );
}

function PlanChangeAction({
  isAdmin,
  hasPlan,
  emphasize,
  onClick,
  chooseLabel,
}: {
  isAdmin: boolean | undefined;
  hasPlan: boolean;
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
