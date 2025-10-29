"use client";
import { Button, SettingCard } from "@unkey/ui";
import type { Stripe } from "stripe";
import { useRouter } from "next/navigation";

export const SubscriptionStatus: React.FC<{
  status: Stripe.Subscription.Status;
  workspaceSlug: string;
}> = (props) => {
  const router = useRouter();

  const statusList = ["incomplete", "incomplete_expired", "unpaid", "past_due"];

  if (statusList.includes(props.status)) {
    return (
      <SettingCard
        title="Payment Required"
        description="There is a problem with your payment. Please resolve it."
        border="both"
        className="border-error-7 bg-error-3"
      >
        <div className="flex justify-end w-full">
          <Button
            variant="primary"
            size="lg"
            onClick={() =>
              router.push(
                `/${props.workspaceSlug}/settings/billing/stripe/portal`
              )
            }
          >
            Open Portal
          </Button>
        </div>
      </SettingCard>
    );
  }
  return null;
};
