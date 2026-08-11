"use client";

import { formatDollars, formatPeriod } from "@/lib/fmt";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import {
  Button,
  InfoTooltip,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemHeader,
  ItemTitle,
  Skeleton,
  toast,
} from "@unkey/ui";
import { useRouter } from "next/navigation";
import { ADMIN_ONLY_TOOLTIP } from "./constants";

type InvoiceCardProps = {
  workspaceSlug: string;
  isAdmin: boolean;
  hasPaymentMethod: boolean;
};

/**
 * What the workspace owes next, as one figure. Each product bills on its own
 * invoice, so the figure sums the previews and the caption names both periods
 * whenever they differ.
 */
export function InvoiceCard({ workspaceSlug, isAdmin, hasPaymentMethod }: InvoiceCardProps) {
  const {
    data: invoice,
    isLoading,
    isError,
  } = trpc.stripe.getUpcomingInvoice.useQuery(undefined, {
    enabled: hasPaymentMethod,
    staleTime: 30_000,
    trpc: { context: { skipBatch: true } },
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
  const caption = isError
    ? "The upcoming invoice could not be loaded."
    : periods.length === 1
      ? periods[0]
      : halves
          .map((row) => `${row.label} ${formatPeriod(row.half.periodStart, row.half.periodEnd)}`)
          .join(" · ");

  // A workspace with nothing subscribed owes nothing, so an empty preview is $0.
  // Only a failed query leaves the amount unknown.
  const totalCents = isError ? null : halves.reduce((total, row) => total + row.half.total, 0);

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
            {totalCents === null ? "—" : formatDollars(totalCents)}
          </span>
        )}
      </div>
    </ItemGroup>
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
