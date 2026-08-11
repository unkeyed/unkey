"use client";

import { DEPLOY_METER_RATE_LABELS } from "@/lib/billing/deployPricing";
import { formatDollars } from "@/lib/fmt";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import {
  Button,
  InfoTooltip,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemFooter,
  ItemGroup,
  ItemHeader,
  ItemSeparator,
  ItemTitle,
  Skeleton,
  toast,
} from "@unkey/ui";
import { useRouter } from "next/navigation";
import { ADMIN_ONLY_TOOLTIP } from "./constants";
import { type DeployInvoice, reconcileDeployInvoice } from "./deploy-invoice";

/** Subscriptions are anchored at 00:00 UTC, so the period reads in UTC too. */
function formatPeriod(startMillis: number, endMillis: number): string {
  const day = (millis: number) =>
    new Date(millis).toLocaleDateString("en-US", { day: "numeric", timeZone: "UTC" });
  const monthYear = (millis: number) =>
    new Date(millis).toLocaleDateString("en-US", {
      month: "long",
      year: "numeric",
      timeZone: "UTC",
    });

  return monthYear(startMillis) === monthYear(endMillis)
    ? `${day(startMillis)} – ${day(endMillis)} ${monthYear(startMillis)}`
    : `${day(startMillis)} ${monthYear(startMillis)} – ${day(endMillis)} ${monthYear(endMillis)}`;
}

function formatRenewalDate(millis: number): string {
  return new Date(millis).toLocaleDateString("en-US", { month: "short", day: "numeric" });
}

type InvoiceCardProps = {
  workspaceSlug: string;
  isAdmin: boolean;
  hasPaymentMethod: boolean;
};

/**
 * What the workspace owes next, as one figure. Each product bills on its own
 * invoice, so the figure sums the previews and the caption names both periods
 * whenever they differ. The Compute breakdown underneath is what that half
 * reconciles to; the API half is a flat fee with nothing to reconcile.
 */
export function InvoiceCard({ workspaceSlug, isAdmin, hasPaymentMethod }: InvoiceCardProps) {
  const { data: invoice, isLoading } = trpc.stripe.getUpcomingInvoice.useQuery(undefined, {
    enabled: hasPaymentMethod,
    staleTime: 30_000,
    trpc: { context: { skipBatch: true } },
  });

  const { data: subscription } = trpc.stripe.getDeploySubscription.useQuery(undefined, {
    staleTime: 30_000,
  });
  const currentPlan = subscription?.plan ?? null;

  const { data: plansData } = trpc.stripe.getDeployPlans.useQuery(undefined, {
    staleTime: 60_000,
    trpc: { context: { skipBatch: true } },
  });
  const { data: deployCredit } = trpc.stripe.getDeployCredit.useQuery(undefined, {
    enabled: Boolean(currentPlan),
    staleTime: 30_000,
    trpc: { context: { skipBatch: true } },
  });
  const { data: usage } = trpc.billing.queryDeployUsage.useQuery(undefined, {
    enabled: Boolean(currentPlan),
    staleTime: 30_000,
    trpc: { context: { skipBatch: true } },
    retry: 1,
  });

  if (!hasPaymentMethod) {
    return <NoPaymentMethod workspaceSlug={workspaceSlug} isAdmin={isAdmin} />;
  }

  const halves = [
    invoice?.deploy ? { label: "Compute", half: invoice.deploy } : null,
    invoice?.api ? { label: "API", half: invoice.api } : null,
  ].filter((row): row is NonNullable<typeof row> => row !== null);

  const periods = [
    ...new Set(halves.map((row) => formatPeriod(row.half.periodStart, row.half.periodEnd))),
  ];
  const caption =
    periods.length === 1
      ? periods[0]
      : halves
          .map((row) => `${row.label} ${formatPeriod(row.half.periodStart, row.half.periodEnd)}`)
          .join(" · ");

  const reconciliation = currentPlan
    ? reconcileDeployInvoice({
        usageCents: usage?.grossCents ?? null,
        projectedUsageCents: usage?.projectedGrossCents ?? null,
        planFeeCents: plansData?.plans.find((p) => p.plan === currentPlan)?.amount ?? null,
        grantedCreditCents: deployCredit?.includedCreditCents ?? null,
        previewTotalCents: invoice?.deploy?.total ?? null,
      })
    : null;

  return (
    <ItemGroup variant="outline">
      <ItemHeader>
        <ItemContent>
          <ItemTitle>Upcoming invoice</ItemTitle>
          {caption ? <ItemDescription>{caption}</ItemDescription> : null}
        </ItemContent>
        <ItemActions>
          <InfoTooltip content={ADMIN_ONLY_TOOLTIP} disabled={isAdmin} asChild>
            <span>
              <Button
                variant="outline"
                size="sm"
                disabled={!isAdmin}
                onClick={() =>
                  // The portal route redirects to Stripe's hosted portal, leaving
                  // the dashboard — open it in a new tab so this page stays put.
                  window.open(
                    routes.settings.stripe.portal({ workspaceSlug }),
                    "_blank",
                    "noopener,noreferrer",
                  )
                }
              >
                Invoices
              </Button>
            </span>
          </InfoTooltip>
        </ItemActions>
      </ItemHeader>

      <div className="px-4 pb-4">
        {isLoading ? (
          <Skeleton className="h-9 w-28" />
        ) : (
          <span className="font-semibold text-3xl text-gray-12 tabular-nums tracking-tight">
            {/* No preview at all is not the same as owing nothing. */}
            {halves.length === 0
              ? "—"
              : formatDollars(halves.reduce((total, row) => total + row.half.total, 0))}
          </span>
        )}
      </div>

      {reconciliation ? (
        <>
          <ItemSeparator />
          <ComputeBreakdown
            invoice={reconciliation}
            renewsAtMillis={invoice?.deploy?.periodEnd ?? null}
          />
        </>
      ) : null}
    </ItemGroup>
  );
}

/**
 * Only the overage and the next plan fee are money owed, so only they carry
 * normal weight. This period's fee and credit balance are context, dimmed and
 * marked paid: they explain where the overage came from without reading as
 * charges, which is what made a prorated period look undercharged. The credit
 * reads as what is LEFT rather than what was applied, because a credit
 * subtracted from a total makes a $0 line look like a missing grant.
 */
function ComputeBreakdown({
  invoice,
  renewsAtMillis,
}: {
  invoice: DeployInvoice;
  renewsAtMillis: number | null;
}) {
  return (
    <ItemFooter className="flex-col items-stretch gap-1.5">
      <span className="text-gray-9">Compute breakdown</span>

      <Line label="Plan fee" note={invoice.feeProrated ? "prorated" : undefined}>
        {formatDollars(invoice.periodFeeCents)} paid
      </Line>
      <Line label="Usage">{formatDollars(invoice.usageCents)}</Line>
      {invoice.includedCreditCents > 0 ? (
        <Line label="Included credit">
          {formatDollars(invoice.creditRemainingCents)} of{" "}
          {formatDollars(invoice.includedCreditCents)} remaining
        </Line>
      ) : null}
      {invoice.includedCreditCents > 0 ? (
        <Line
          label="Overage"
          note="usage past credit"
          className="mt-1 border-grayA-3 border-t pt-2"
        >
          {formatDollars(invoice.overageCents)}
        </Line>
      ) : null}
      <Line label="Next plan fee" note="month ahead">
        {formatDollars(invoice.nextPlanFeeCents)}
      </Line>

      <div className="mt-1 flex items-baseline justify-between gap-4 border-grayA-3 border-t pt-2">
        <span className="text-[13px] text-gray-12">
          <InfoTooltip
            asChild
            position={{ side: "top", align: "start" }}
            content={
              <div className="flex max-w-[240px] flex-col gap-2 text-[12px]">
                <div className="flex flex-col gap-0.5">
                  <p className="font-medium text-gray-12">How this is calculated</p>
                  <p className="text-gray-11">
                    This period's {formatDollars(invoice.periodFeeCents)} fee is already invoiced
                    and covers {formatDollars(invoice.includedCreditCents)} of usage. The next
                    invoice charges what your usage went past that, plus the coming period's fee.
                  </p>
                  <p className="text-gray-11 tabular-nums">
                    {formatDollars(invoice.overageCents)} +{" "}
                    {formatDollars(invoice.nextPlanFeeCents)} ={" "}
                    {formatDollars(invoice.nextInvoiceCents)}
                  </p>
                </div>
                <div className="flex flex-col gap-1 border-grayA-4 border-t pt-2">
                  <p className="font-medium text-gray-12">Usage rates</p>
                  <ul className="flex flex-col gap-0.5">
                    {DEPLOY_METER_RATE_LABELS.map((rate) => (
                      <li key={rate.label} className="flex justify-between gap-3 text-gray-11">
                        <span>{rate.label}</span>
                        <span className="tabular-nums">{rate.rate}</span>
                      </li>
                    ))}
                  </ul>
                </div>
              </div>
            }
          >
            <span className="cursor-help underline decoration-dotted decoration-grayA-6 underline-offset-2">
              Next invoice
              {renewsAtMillis !== null ? ` · ${formatRenewalDate(renewsAtMillis)}` : ""}
            </span>
          </InfoTooltip>
          {invoice.projectedNextInvoiceCents !== null ? (
            <span className="ml-1.5 text-[12px] text-gray-9">
              (~{formatDollars(invoice.projectedNextInvoiceCents)} projected)
            </span>
          ) : null}
        </span>
        <span className="font-medium text-[13px] text-gray-12 tabular-nums">
          {formatDollars(invoice.nextInvoiceCents)}
        </span>
      </div>

      <p className="text-gray-9">
        This period's {formatDollars(invoice.periodFeeCents)} fee is already paid. Total cost for
        this period is {formatDollars(invoice.periodTotalCents)}.
      </p>
    </ItemFooter>
  );
}

function Line({
  label,
  note,
  className,
  children,
}: {
  label: string;
  note?: string;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <div className={`flex items-baseline justify-between gap-4 ${className ?? ""}`}>
      <span className="text-gray-10">
        {label}
        {note ? <span className="ml-1.5 text-gray-9">{note}</span> : null}
      </span>
      <span className="text-gray-9 tabular-nums">{children}</span>
    </div>
  );
}

/**
 * The one action that unblocks everything below it, so it takes the card rather
 * than sitting beside a figure the workspace cannot yet have.
 */
function NoPaymentMethod({
  workspaceSlug,
  isAdmin,
}: {
  workspaceSlug: string;
  isAdmin: boolean;
}) {
  const router = useRouter();
  const trpcUtils = trpc.useUtils();

  // Dev-only: one-click seed a Stripe test customer + 4242 card so local runs
  // skip the checkout round-trip. The mutation is gated on STRIPE_DEV_TEST_CLOCK
  // server-side; the button only renders in dev.
  const seedStripe = trpc.stripe.seedTestCustomer.useMutation({
    onSuccess: async () => {
      toast.success("Seeded a Stripe test customer with the 4242 card");
      await trpcUtils.invalidate();
      router.refresh();
    },
    onError: (err) => toast.error(err.message),
  });

  return (
    <ItemGroup variant="outline">
      <ItemHeader>
        <ItemContent>
          <ItemTitle>No payment method</ItemTitle>
          <ItemDescription>
            Add one to subscribe. Each product bills on its own invoice.
          </ItemDescription>
        </ItemContent>
        <ItemActions>
          {process.env.NODE_ENV === "development" ? (
            <Button
              variant="outline"
              size="sm"
              disabled={!isAdmin || seedStripe.isLoading}
              onClick={() => seedStripe.mutate()}
              title="Dev only: create a Stripe test customer with your email and the 4242 test card"
            >
              {seedStripe.isLoading ? "Seeding..." : "Seed test card"}
            </Button>
          ) : null}
          <InfoTooltip content={ADMIN_ONLY_TOOLTIP} disabled={isAdmin} asChild>
            <span>
              <Button
                variant="primary"
                size="sm"
                disabled={!isAdmin}
                onClick={() =>
                  router.push(routes.settings.stripe.checkout({ workspaceSlug, intent: "payment" }))
                }
              >
                Add payment method
              </Button>
            </span>
          </InfoTooltip>
        </ItemActions>
      </ItemHeader>
    </ItemGroup>
  );
}
