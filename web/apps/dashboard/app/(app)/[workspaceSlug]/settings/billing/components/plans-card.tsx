"use client";

import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import type { inferRouterOutputs } from "@trpc/server";
import {
  Button,
  ItemActions,
  ItemContent,
  ItemGroup,
  ItemHeader,
  ItemSeparator,
  ItemTitle,
  toast,
} from "@unkey/ui";
import { useRouter } from "next/navigation";
import { AdminGate } from "./admin-gate";
import { currentApiProduct } from "./api-plan";
import { ApiPlanRow } from "./api-plan-row";
import { ComputePlanRow } from "./compute-plan-row";
import { PlanTableHeader } from "./plan-table-row";

type BillingInfo = inferRouterOutputs<Router>["stripe"]["getBillingInfo"];

type PlansCardProps = {
  isAdmin: boolean | undefined;
  hasPaymentMethod: boolean;
  workspaceSlug: string;
  products: BillingInfo["products"];
  subscription?: BillingInfo["subscription"];
  currentProductId?: BillingInfo["currentProductId"];
  checkoutIntent: "compute" | "api" | null;
};

export function PlansCard({
  isAdmin,
  hasPaymentMethod,
  workspaceSlug,
  products,
  subscription,
  currentProductId,
  checkoutIntent,
}: PlansCardProps) {
  const router = useRouter();
  const trpcUtils = trpc.useUtils();
  const { data: deploySubscription } = trpc.stripe.getDeploySubscription.useQuery(undefined, {
    staleTime: 30_000,
  });

  const seedStripe = trpc.stripe.seedTestCustomer.useMutation({
    onSuccess: async () => {
      toast.success("Seeded a Stripe test customer with the 4242 card");
      await trpcUtils.invalidate();
      router.refresh();
    },
    onError: (err) => toast.error(err.message),
  });

  const hasPaidApiPlan = Boolean(currentApiProduct({ products, subscription, currentProductId }));
  const emphasize = !deploySubscription?.plan && !hasPaidApiPlan;

  return (
    <ItemGroup variant="outline">
      <ItemHeader>
        <ItemContent>
          <ItemTitle>Plans</ItemTitle>
        </ItemContent>
        <ItemActions>
          {hasPaymentMethod ? (
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
                  View invoices
                </Button>
              )}
            </AdminGate>
          ) : (
            <>
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
                      router.push(
                        routes.settings.stripe.checkout({ workspaceSlug, intent: "payment" }),
                      )
                    }
                  >
                    Add payment method
                  </Button>
                )}
              </AdminGate>
            </>
          )}
        </ItemActions>
      </ItemHeader>

      <ItemSeparator />
      <PlanTableHeader />
      <ItemSeparator />
      <ComputePlanRow
        isAdmin={isAdmin}
        hasPaymentMethod={hasPaymentMethod}
        workspaceSlug={workspaceSlug}
        emphasize={emphasize}
        autoOpenPlanModal={checkoutIntent === "compute" && hasPaymentMethod}
      />

      <ItemSeparator />
      <ApiPlanRow
        isAdmin={isAdmin}
        hasPaymentMethod={hasPaymentMethod}
        workspaceSlug={workspaceSlug}
        emphasize={emphasize}
        products={products}
        subscription={subscription}
        currentProductId={currentProductId}
        autoOpenPlanModal={checkoutIntent === "api" && hasPaymentMethod}
      />
    </ItemGroup>
  );
}
