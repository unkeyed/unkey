"use client";

import { Button, Dialog, DialogContent, DialogDescription, DialogTitle } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

type PaymentFailureModalProps = {
  gracePeriodEndsAt: number;
  workspaceSlug: string;
};

const DISMISS_KEY = "payment-failure-dismissed";
const DISMISS_DURATION = 24 * 60 * 60 * 1000; // 24 hours

export const PaymentFailureModal = ({
  gracePeriodEndsAt,
  workspaceSlug,
}: PaymentFailureModalProps) => {
  const [isOpen, setIsOpen] = useState(false);
  const router = useRouter();

  useEffect(() => {
    try {
      const dismissedAt = localStorage.getItem(DISMISS_KEY);
      const now = Date.now();

      if (dismissedAt) {
        const dismissed = Number.parseInt(dismissedAt, 10);
        if (now - dismissed < DISMISS_DURATION) {
          return;
        }
      }
    } catch {
      // localStorage may be unavailable in private browsing
    }

    setIsOpen(true);
  }, []);

  const handleDismiss = () => {
    try {
      localStorage.setItem(DISMISS_KEY, Date.now().toString());
    } catch {
      // localStorage may be unavailable
    }
    setIsOpen(false);
  };

  const handleUpdatePayment = () => {
    handleDismiss();
    router.push(`/${workspaceSlug}/settings/billing/stripe/portal`);
  };

  const daysRemaining = Math.max(0, Math.ceil((gracePeriodEndsAt - Date.now()) / (24 * 60 * 60 * 1000)));

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogContent className="max-w-md">
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-2">
            <DialogTitle>Payment Failed</DialogTitle>
            <DialogDescription>
              Your payment method failed. You have {daysRemaining} day
              {daysRemaining !== 1 ? "s" : ""} remaining to update your payment method before your
              subscription is downgraded to the free tier.
            </DialogDescription>
          </div>

          <div className="flex flex-col gap-2">
            <Button variant="primary" onClick={handleUpdatePayment}>
              Update Payment Method
            </Button>
            <Button variant="ghost" onClick={handleDismiss}>
              Remind Me Later
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
};
