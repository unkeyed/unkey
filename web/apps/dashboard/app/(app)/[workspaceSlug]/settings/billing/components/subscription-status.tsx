"use client";
import { usePayInvoice } from "@/hooks/use-payment-issues";
import { needsPaymentAction } from "@/lib/stripe/subscriptionUtils";
import { Button, SettingCard } from "@unkey/ui";
import type { Stripe } from "stripe";
import { AdminGate } from "./admin-gate";

type PaymentKind = "expired" | "payable";

const PAYMENT_KIND: Record<PaymentKind, { description: string; showPay: boolean }> = {
  expired: {
    description: "The previous payment attempt expired. Choose your plan again to retry.",
    showPay: false,
  },
  payable: {
    description: "Complete the payment in Stripe to activate or restore your plan.",
    showPay: true,
  },
};

function paymentKind(status: Stripe.Subscription.Status): PaymentKind | null {
  if (status === "incomplete_expired") {
    return "expired";
  }
  return needsPaymentAction(status) ? "payable" : null;
}

export function SubscriptionStatus({
  status,
  isAdmin,
}: {
  status: Stripe.Subscription.Status;
  isAdmin: boolean | undefined;
}) {
  const payInvoice = usePayInvoice();
  const kind = paymentKind(status);

  if (kind === null) {
    return null;
  }

  const { description, showPay } = PAYMENT_KIND[kind];

  return (
    <SettingCard
      title="Payment Required"
      description={description}
      border="both"
      className="border-error-7 bg-error-3"
    >
      {showPay ? (
        <div className="flex justify-end w-full">
          <AdminGate isAdmin={isAdmin}>
            {(disabled) => (
              <Button
                variant="primary"
                size="lg"
                disabled={disabled || payInvoice.busy}
                loading={payInvoice.busy}
                onClick={() => payInvoice.pay("api")}
              >
                Complete payment
              </Button>
            )}
          </AdminGate>
        </div>
      ) : null}
    </SettingCard>
  );
}
