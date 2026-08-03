"use client";
import { trpc } from "@/lib/trpc/client";
import { Button, SettingCard, toast } from "@unkey/ui";
import type { Stripe } from "stripe";

export const SubscriptionStatus: React.FC<{
  status: Stripe.Subscription.Status;
}> = (props) => {
  const completePayment = trpc.stripe.getSubscriptionPaymentUrl.useMutation({
    onSuccess: ({ paymentUrl }) => window.location.assign(paymentUrl),
    onError: (error) => toast.error(error.message),
  });

  const statusList = ["incomplete", "incomplete_expired", "unpaid", "past_due"];

  if (statusList.includes(props.status)) {
    return (
      <SettingCard
        title="Payment Required"
        description={
          props.status === "incomplete_expired"
            ? "The previous payment attempt expired. Choose your plan again to retry."
            : "Complete the payment in Stripe to activate or restore your plan."
        }
        border="both"
        className="border-error-7 bg-error-3"
      >
        {props.status !== "incomplete_expired" ? (
          <div className="flex justify-end w-full">
            <Button
              variant="primary"
              size="lg"
              loading={completePayment.isLoading}
              onClick={() => completePayment.mutate()}
            >
              Complete payment
            </Button>
          </div>
        ) : null}
      </SettingCard>
    );
  }
  return null;
};
