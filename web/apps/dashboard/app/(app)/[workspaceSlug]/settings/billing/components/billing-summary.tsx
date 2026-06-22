"use client";

import { formatDollars } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import { Button, InfoTooltip } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { ADMIN_ONLY_TOOLTIP } from "./constants";

function formatPeriodDate(millis: number): string {
  return new Date(millis).toLocaleDateString("en-US", { month: "short", day: "numeric" });
}

type BillingSummaryProps = {
  workspaceSlug: string;
  isAdmin: boolean;
  hasPaymentMethod: boolean;
};

/**
 * The headline strip: current billing period and what the next invoice will
 * be, with the Stripe portal (payment methods, past invoices) one click away.
 * Without a payment method it collapses into the add-payment-method CTA, the
 * one action that unblocks everything below it.
 */
export const BillingSummary: React.FC<BillingSummaryProps> = ({
  workspaceSlug,
  isAdmin,
  hasPaymentMethod,
}) => {
  const router = useRouter();
  const { data: invoice, isLoading } = trpc.stripe.getUpcomingInvoice.useQuery(undefined, {
    enabled: hasPaymentMethod,
    staleTime: 30_000,
  });

  if (!hasPaymentMethod) {
    return (
      <div className="flex w-full items-center justify-between gap-4 rounded-xl border border-grayA-4 bg-white px-5 py-4 dark:bg-black">
        <div>
          <p className="font-medium text-gray-12 text-sm">No payment method</p>
          <p className="text-[13px] text-gray-10">
            One payment method covers Compute and API plans on a single invoice.
          </p>
        </div>
        <InfoTooltip content={ADMIN_ONLY_TOOLTIP} disabled={isAdmin} asChild>
          <span>
            <Button
              variant="primary"
              size="md"
              disabled={!isAdmin}
              onClick={() =>
                router.push(`/${workspaceSlug}/settings/billing/stripe/checkout?intent=payment`)
              }
            >
              Add payment method
            </Button>
          </span>
        </InfoTooltip>
      </div>
    );
  }

  return (
    <div className="flex w-full items-center justify-between gap-4 rounded-xl border border-grayA-4 bg-white px-5 py-4 dark:bg-black">
      {/* Both columns share one type ramp and start at the top, so the two
          labels and the two values each sit on a common baseline. */}
      <div className="flex items-start gap-10">
        <div className="flex flex-col gap-1">
          <p className="text-[13px] text-gray-10 leading-4">Current billing cycle</p>
          {isLoading ? (
            <div className="h-5 w-24 animate-pulse rounded bg-grayA-3" />
          ) : invoice ? (
            <p className="font-medium text-gray-12 text-sm leading-5 tabular-nums">
              {formatPeriodDate(invoice.periodStart)} – {formatPeriodDate(invoice.periodEnd)}
            </p>
          ) : (
            <p className="font-medium text-gray-12 text-sm leading-5">—</p>
          )}
        </div>
        <div className="flex flex-col gap-1">
          <p className="text-[13px] text-gray-10 leading-4">Upcoming invoice</p>
          {isLoading ? (
            <div className="h-5 w-16 animate-pulse rounded bg-grayA-3" />
          ) : (
            <p className="font-medium text-gray-12 text-sm leading-5 tabular-nums">
              {invoice ? formatDollars(invoice.total) : formatDollars(0)}
              {/* The preview already contains usage reported so far, but it
                  keeps accruing until the period closes. */}
              <span className="ml-1.5 font-normal text-gray-9 text-xs">
                + usage until the period ends
              </span>
            </p>
          )}
        </div>
      </div>
      <InfoTooltip content={ADMIN_ONLY_TOOLTIP} disabled={isAdmin} asChild>
        <span>
          <Button
            variant="outline"
            size="md"
            disabled={!isAdmin}
            onClick={() =>
              // The portal route redirects to Stripe's hosted portal, leaving
              // the dashboard — open it in a new tab so this page stays put.
              window.open(
                `/${workspaceSlug}/settings/billing/stripe/portal`,
                "_blank",
                "noopener,noreferrer",
              )
            }
          >
            Invoices & payment
          </Button>
        </span>
      </InfoTooltip>
    </div>
  );
};
