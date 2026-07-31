"use client";

import { formatDollars } from "@/lib/fmt";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { Button, InfoTooltip, toast } from "@unkey/ui";
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
    trpc: { context: { skipBatch: true } },
  });

  // Dev-only: one-click seed a Stripe test customer + 4242 card so local runs
  // skip the checkout round-trip and re-entering sandbox card data. The mutation
  // is gated on STRIPE_DEV_TEST_CLOCK server-side; the button only renders in dev.
  const trpcUtils = trpc.useUtils();
  const seedStripe = trpc.stripe.seedTestCustomer.useMutation({
    onSuccess: async () => {
      toast.success("Seeded a Stripe test customer with the 4242 card");
      await trpcUtils.invalidate();
      router.refresh();
    },
    onError: (err) => toast.error(err.message),
  });

  if (!hasPaymentMethod) {
    return (
      <div className="flex w-full items-center justify-between gap-4 rounded-lg border border-grayA-4 bg-white px-5 py-4 dark:bg-black">
        <div>
          <p className="font-medium text-gray-12 text-sm">No payment method</p>
          <p className="text-[13px] text-gray-10">
            Add one to subscribe. Each product bills on its own invoice.
          </p>
        </div>
        <div className="flex items-center gap-2">
          {process.env.NODE_ENV === "development" ? (
            <Button
              variant="outline"
              size="md"
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
                size="md"
                disabled={!isAdmin}
                onClick={() =>
                  router.push(routes.settings.stripe.checkout({ workspaceSlug, intent: "payment" }))
                }
              >
                Add payment method
              </Button>
            </span>
          </InfoTooltip>
        </div>
      </div>
    );
  }

  // Each product bills on its own invoice now, so render a row per active
  // product. Only the Compute row carries metered usage that keeps accruing.
  const rows = [
    invoice?.api ? { label: "API", half: invoice.api, showUsage: false } : null,
    invoice?.deploy ? { label: "Compute", half: invoice.deploy, showUsage: true } : null,
  ].filter((row): row is NonNullable<typeof row> => row !== null);

  return (
    <div className="flex w-full flex-col gap-4 rounded-lg border border-grayA-4 bg-white px-5 py-4 dark:bg-black">
      <div className="flex items-center justify-between gap-4">
        <div>
          <p className="font-medium text-gray-12 text-sm">Upcoming invoices</p>
          <p className="text-[13px] text-gray-10">Each product bills on its own invoice.</p>
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
                  routes.settings.stripe.portal({ workspaceSlug }),
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

      {isLoading ? (
        <div className="h-5 w-40 animate-pulse rounded bg-grayA-3" />
      ) : rows.length > 0 ? (
        <div className="flex flex-col gap-2">
          {rows.map((row) => (
            <div key={row.label} className="flex items-baseline justify-between gap-4">
              <div className="flex items-baseline gap-3">
                <span className="w-16 font-medium text-gray-12 text-[13px]">{row.label}</span>
                <span className="text-[13px] text-gray-10 tabular-nums">
                  {formatPeriodDate(row.half.periodStart)} – {formatPeriodDate(row.half.periodEnd)}
                </span>
              </div>
              <p className="font-medium text-gray-12 text-sm leading-5 tabular-nums">
                {formatDollars(row.half.total)}
                {row.showUsage ? (
                  <span className="ml-1.5 font-normal text-gray-9 text-xs">
                    + usage until the period ends
                  </span>
                ) : null}
              </p>
            </div>
          ))}
        </div>
      ) : (
        <p className="font-medium text-gray-12 text-sm">—</p>
      )}
    </div>
  );
};
