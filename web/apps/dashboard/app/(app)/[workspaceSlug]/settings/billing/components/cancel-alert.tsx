"use client";
import { trpc } from "@/lib/trpc/client";
import { SettingsZone, SettingsZoneRow, toast } from "@unkey/ui";
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
  if (timeRemaining <= 0) {
    return null;
  }

  return (
    <SettingsZone variant="warning" title="Cancellation Scheduled">
      <SettingsZoneRow
        title="Subscription ending"
        description={
          <>
            Your subscription ends in
            <span className="text-warning-12 font-medium">
              {" "}
              {ms(timeRemaining, { long: true })}
            </span>{" "}
            on{" "}
            <span className="text-warning-12 font-medium">
              {new Date(props.cancelAt).toLocaleDateString()}
            </span>
            .
          </>
        }
        action={{
          label: "Resubscribe",
          onClick: () => uncancelSubscription.mutate(),
          loading: uncancelSubscription.isLoading,
          disabled: uncancelSubscription.isLoading,
        }}
      />
    </SettingsZone>
  );
};
