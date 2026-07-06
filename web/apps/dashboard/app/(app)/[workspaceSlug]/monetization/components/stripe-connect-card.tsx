"use client";
import { trpc } from "@/lib/trpc/client";
import { Button, InfoTooltip, SettingCard, toast } from "@unkey/ui";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useRef } from "react";

const ADMIN_ONLY_TOOLTIP = "Admin access required to manage billing";

/**
 * Stripe Connect card for end-user billing: the customer connects their own
 * Stripe account through Stripe-hosted onboarding, so Unkey can push their
 * end-users' priced usage to it each billing period. The customer's account
 * stays merchant-of-record; Unkey never holds the funds.
 */
export const StripeConnectCard: React.FC<{ isAdmin: boolean }> = ({ isAdmin }) => {
  const router = useRouter();
  const searchParams = useSearchParams();
  const utils = trpc.useUtils();

  const { data, isLoading } = trpc.billing.stripeConnect.get.useQuery(undefined, {
    staleTime: 30_000,
  });
  const status = data?.status ?? "none";

  const startOnboarding = trpc.billing.stripeConnect.startOnboarding.useMutation({
    onSuccess: ({ url }) => {
      window.location.href = url;
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });

  const finishOnboarding = trpc.billing.stripeConnect.finishOnboarding.useMutation({
    onSuccess: (res) => {
      utils.billing.stripeConnect.get.invalidate();
      if (res.status === "linked") {
        toast.success("Stripe account connected. End-user billing is now active.");
      } else if (res.status === "pending") {
        toast.info("Stripe onboarding is not finished yet. Resume it to complete the connection.");
      }
    },
  });

  const unlink = trpc.billing.stripeConnect.unlink.useMutation({
    onSuccess: () => {
      utils.billing.stripeConnect.get.invalidate();
      toast.success("Stripe account disconnected.");
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });

  // Returning from Stripe-hosted onboarding (?connect=return|refresh):
  // reconcile against Stripe once, then clean the URL.
  const reconciled = useRef(false);
  const connectParam = searchParams?.get("connect");
  useEffect(() => {
    if (!connectParam || reconciled.current || !isAdmin) {
      return;
    }
    reconciled.current = true;
    finishOnboarding.mutate();
    const url = new URL(window.location.href);
    url.searchParams.delete("connect");
    router.replace(url.pathname + url.search);
  }, [connectParam, isAdmin, finishOnboarding, router]);

  const busy =
    isLoading || startOnboarding.isLoading || finishOnboarding.isLoading || unlink.isLoading;

  return (
    <SettingCard
      title="End-user billing"
      description={
        status === "linked"
          ? "Your Stripe account is connected. Your end-users' priced usage is pushed to it at each period close."
          : status === "pending"
            ? "Stripe onboarding was started but not finished. Resume it to activate end-user billing."
            : "Connect your Stripe account to bill your own users for their usage. You stay the merchant of record."
      }
    >
      <div className="w-full flex h-full items-center justify-end gap-4">
        {status === "linked" ? (
          <InfoTooltip content={ADMIN_ONLY_TOOLTIP} disabled={isAdmin} asChild>
            <span>
              <Button
                variant="outline"
                className="py-2 px-3 text-gray-12 font-medium text-sm bg-grayA-2 hover:bg-grayA-3"
                aria-label="Disconnect Stripe account"
                disabled={!isAdmin || busy}
                onClick={() => unlink.mutate()}
              >
                Disconnect
              </Button>
            </span>
          </InfoTooltip>
        ) : (
          <InfoTooltip content={ADMIN_ONLY_TOOLTIP} disabled={isAdmin} asChild>
            <span>
              <Button
                variant="primary"
                className="py-2 px-3 font-medium text-sm"
                aria-label="Connect Stripe account"
                disabled={!isAdmin || busy}
                loading={startOnboarding.isLoading}
                onClick={() => startOnboarding.mutate()}
              >
                {status === "pending" ? "Resume onboarding" : "Connect Stripe"}
              </Button>
            </span>
          </InfoTooltip>
        )}
      </div>
    </SettingCard>
  );
};
