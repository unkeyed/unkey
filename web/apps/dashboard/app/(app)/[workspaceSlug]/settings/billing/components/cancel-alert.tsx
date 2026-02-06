"use client";
import { trpc } from "@/lib/trpc/client";
import { Button, SettingCard, toast } from "@unkey/ui";
import ms from "ms";
import { useRouter } from "next/navigation";

export const CancelAlert: React.FC<{ cancelAt?: number }> = (props) => {
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
  // If cancellation date has passed, don't show the alert
  if (timeRemaining <= 0) {
    return null;
  }

  return (
    <SettingCard
      title="Cancellation scheduled"
      description={
        <p>
          Your subscription ends in
          <span className="text-accent-12"> {ms(timeRemaining, { long: true })}</span> on{" "}
          <span className="text-accent-12">{new Date(props.cancelAt).toLocaleDateString()}</span>.
        </p>
      }
      border="both"
      className="border-warning-7 bg-warning-2 w-full"
      contentWidth="w-full lg:w-[320px]"
    >
      <div className="w-full flex h-full items-center justify-end gap-4">
        <Button
          variant="primary"
          loading={uncancelSubscription.isLoading}
          disabled={uncancelSubscription.isLoading}
          onClick={() => uncancelSubscription.mutate()}
        >
          Resubscribe
        </Button>
      </div>
    </SettingCard>
  );
};
