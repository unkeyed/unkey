"use client";
import { Button, SettingCard } from "@unkey/ui";
import { useRouter } from "next/navigation";
import type { Stripe } from "stripe";

export const SubscriptionStatus: React.FC<{
  status: Stripe.Subscription.Status;
  workspaceSlug: string;
  gracePeriodEndsAt?: number;
}> = (props) => {
  const router = useRouter();

  const statusList = ["incomplete", "incomplete_expired", "unpaid", "past_due"];

  if (statusList.includes(props.status)) {
    const daysRemaining = props.gracePeriodEndsAt
      ? Math.max(0, Math.ceil((props.gracePeriodEndsAt - Date.now()) / (24 * 60 * 60 * 1000)))
      : 0;

    return (
      <SettingCard
        title="Payment Failed"
        description={
          props.gracePeriodEndsAt
            ? `Your payment method failed. You have ${daysRemaining} day${daysRemaining !== 1 ? "s" : ""} remaining to update your payment method before your subscription is downgraded to the free tier.`
            : "There is a problem with your payment. Please resolve it."
        }
        border="both"
        className="border-error-7 bg-error-3"
      >
        <div className="flex justify-end w-full">
          <Button
            variant="primary"
            size="lg"
            onClick={() => router.push(`/${props.workspaceSlug}/settings/billing/stripe/portal`)}
          >
            Update Payment Method
          </Button>
        </div>
      </SettingCard>
    );
  }
  return null;
};
