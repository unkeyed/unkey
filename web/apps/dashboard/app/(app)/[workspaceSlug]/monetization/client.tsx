"use client";
import { trpc } from "@/lib/trpc/client";
import { SettingCardGroup } from "@unkey/ui";
import { AttachBillingCard } from "./components/attach-billing-card";
import { BillableResourcesCard } from "./components/billable-resources-card";
import { RateCardsCard } from "./components/rate-cards-card";
import { StripeConnectCard } from "./components/stripe-connect-card";

/**
 * Monetization: a customer billing their own end-users from Unkey usage.
 * Distinct from Settings > Billing (Unkey charging the customer). Configures
 * the connected Stripe account, rate cards, which keyspaces/namespaces are
 * billed, and attaches individual end-users to Stripe customers.
 */
export const MonetizationClient: React.FC = () => {
  // Mirrors the server-side requireWorkspaceAdmin gate purely for UX, so
  // non-admins see a clear affordance rather than a request that FORBIDs.
  const { data: currentUser } = trpc.user.getCurrentUser.useQuery();
  const isAdmin = currentUser?.role === "admin";

  return (
    <div className="flex w-full flex-col items-center py-6">
      <div className="flex w-full max-w-4xl flex-col gap-6 px-4">
        <SettingCardGroup>
          <StripeConnectCard isAdmin={isAdmin} />
        </SettingCardGroup>

        <RateCardsCard isAdmin={isAdmin} />

        <BillableResourcesCard isAdmin={isAdmin} />

        <AttachBillingCard isAdmin={isAdmin} />
      </div>
    </div>
  );
};
