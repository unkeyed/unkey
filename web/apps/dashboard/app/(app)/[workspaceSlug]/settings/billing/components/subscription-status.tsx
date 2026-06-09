"use client";
import { Button } from "@unkey/ui";
import { useRouter } from "next/navigation";
import type { Stripe } from "stripe";
import { billingButton } from "./billing-card";

export const SubscriptionStatus: React.FC<{
  status: Stripe.Subscription.Status;
  workspaceSlug: string;
}> = (props) => {
  const router = useRouter();

  const statusList = ["incomplete", "incomplete_expired", "unpaid", "past_due"];

  if (statusList.includes(props.status)) {
    return (
      <div className="flex w-full flex-col gap-3 border border-error-7 bg-error-2 px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex flex-col gap-1">
          <span className="font-mono text-[11px] text-error-11 uppercase tracking-wider">
            Payment required
          </span>
          <p className="font-medium text-gray-12 text-sm">There is a problem with your payment.</p>
          <p className="text-[13px] text-gray-10 leading-snug">
            Open the billing portal to resolve it.
          </p>
        </div>
        <Button
          variant="primary"
          size="lg"
          className={`shrink-0 ${billingButton}`}
          onClick={() => router.push(`/${props.workspaceSlug}/settings/billing/stripe/portal`)}
        >
          Open Portal
        </Button>
      </div>
    );
  }
  return null;
};
