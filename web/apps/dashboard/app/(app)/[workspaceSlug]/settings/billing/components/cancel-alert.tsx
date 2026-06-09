"use client";
import { formatMs } from "@/lib/ms";
import { trpc } from "@/lib/trpc/client";
import { Button, toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { billingButton } from "./billing-card";

export const CancelAlert: React.FC<{
  cancelAt?: number;
  disabled?: boolean;
  disabledReason?: string;
}> = (props) => {
  const router = useRouter();
  const trpcUtils = trpc.useUtils();
  const uncancelSubscription = trpc.stripe.uncancelSubscription.useMutation({
    onSuccess: async () => {
      // Revalidate helper: invalidate AND explicitly refetch to ensure UI updates
      await Promise.all([
        trpcUtils.workspace.getCurrent.invalidate(),
        trpcUtils.billing.queryUsage.invalidate(),
        trpcUtils.stripe.getBillingInfo.invalidate(),
        trpcUtils.workspace.getCurrent.refetch(),
        trpcUtils.stripe.getBillingInfo.refetch(),
      ]);
      router.refresh();
      toast.info("Subscription resumed");
    },
    onError: (err) => {
      toast.error("Failed to resume subscription. Please try again or contact support@unkey.com.");
      console.error("Subscription resumption error:", err);
    },
  });

  if (!props.cancelAt) {
    return null;
  }

  const timeRemaining = props.cancelAt - Date.now();
  if (timeRemaining <= 0) {
    return null;
  }

  const description =
    props.disabled && props.disabledReason ? (
      props.disabledReason
    ) : (
      <>
        Your subscription ends in
        <span className="text-warning-12 font-medium">
          {" "}
          {formatMs(timeRemaining, { long: true })}
        </span>{" "}
        on{" "}
        <span className="text-warning-12 font-medium">
          {new Date(props.cancelAt).toLocaleDateString()}
        </span>
        .
      </>
    );

  return (
    <div className="flex w-full flex-col gap-3 border border-warning-7 bg-warning-2 px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex flex-col gap-1">
        <span className="font-mono text-[11px] text-warning-11 uppercase tracking-wider">
          Cancellation scheduled
        </span>
        <p className="font-medium text-gray-12 text-sm">Subscription ending</p>
        <p className="text-[13px] text-gray-10 leading-snug">{description}</p>
      </div>
      <Button
        variant="primary"
        color="warning"
        size="lg"
        className={`shrink-0 ${billingButton}`}
        loading={uncancelSubscription.isLoading}
        disabled={uncancelSubscription.isLoading || Boolean(props.disabled)}
        onClick={() => uncancelSubscription.mutate()}
      >
        Resubscribe
      </Button>
    </div>
  );
};
