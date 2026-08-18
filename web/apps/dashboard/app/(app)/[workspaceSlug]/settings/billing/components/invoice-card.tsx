"use client";

import { formatPeriod, formatPrice } from "@/lib/fmt";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import {
  Button,
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
import { AdminGate } from "./admin-gate";

type InvoiceCardProps = {
  workspaceSlug: string;
  isAdmin: boolean | undefined;
  hasPaymentMethod: boolean;
};

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

  const totalCents = isError ? null : halves.reduce((total, row) => total + row.half.total, 0);

  return (
    <ItemGroup variant="outline">
      <ItemHeader>
        <ItemContent>
          <ItemTitle>Upcoming invoice</ItemTitle>
          {caption ? <ItemDescription>{caption}</ItemDescription> : null}
        </ItemContent>
        <ItemActions>
          <AdminGate isAdmin={isAdmin}>
            {(disabled) => (
              <Button
                variant="outline"
                size="sm"
                disabled={disabled}
                onClick={() =>
                  window.open(
                    routes.settings.stripe.portal({ workspaceSlug }),
                    "_blank",
                    "noopener,noreferrer",
                  )
                }
              >
                Invoices
              </Button>
            )}
          </AdminGate>
        </ItemActions>
      </ItemHeader>

      <div className="px-4 pb-4">
        {isLoading ? (
          <Skeleton className="h-9 w-28" />
        ) : (
          <span className="font-semibold text-3xl text-gray-12 tabular-nums tracking-tight">
            {totalCents === null ? "—" : formatPrice(totalCents)}
          </span>
        )}
      </div>
    </ItemGroup>
  );
}

function NoPaymentMethod({
  workspaceSlug,
  isAdmin,
}: {
  workspaceSlug: string;
  isAdmin: boolean | undefined;
}) {
  const router = useRouter();
  const trpcUtils = trpc.useUtils();

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
              disabled={isAdmin !== true || seedStripe.isLoading}
              onClick={() => seedStripe.mutate()}
              title="Dev only: create a Stripe test customer with your email and the 4242 test card"
            >
              {seedStripe.isLoading ? "Seeding..." : "Seed test card"}
            </Button>
          ) : null}
          <AdminGate isAdmin={isAdmin}>
            {(disabled) => (
              <Button
                variant="primary"
                size="sm"
                disabled={disabled}
                onClick={() =>
                  router.push(routes.settings.stripe.checkout({ workspaceSlug, intent: "payment" }))
                }
              >
                Add payment method
              </Button>
            )}
          </AdminGate>
        </ItemActions>
      </ItemHeader>
    </ItemGroup>
  );
}
