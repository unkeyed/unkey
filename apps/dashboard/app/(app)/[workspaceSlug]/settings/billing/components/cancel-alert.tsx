"use client";
import { useTRPC } from "@/lib/trpc/client";
import { Button, SettingCard, toast } from "@unkey/ui";
import ms from "ms";
import { useRouter } from "next/navigation";

import { useMutation } from "@tanstack/react-query";
import { useQueryClient } from "@tanstack/react-query";

export const CancelAlert: React.FC<{ cancelAt?: number }> = (props) => {
  const trpc = useTRPC();
  const router = useRouter();
  const queryClient = useQueryClient();
  const uncancelSubscription = useMutation(
    trpc.stripe.uncancelSubscription.mutationOptions({
      onSuccess: async () => {
        // Revalidate helper: invalidate AND explicitly refetch to ensure UI updates
        await Promise.all([
          queryClient.invalidateQueries(trpc.workspace.getCurrent.pathFilter()),
          queryClient.invalidateQueries(trpc.billing.queryUsage.pathFilter()),
          queryClient.invalidateQueries(trpc.stripe.getBillingInfo.pathFilter()),
          queryClient.refetchQueries(trpc.workspace.getCurrent.pathFilter()),
          queryClient.refetchQueries(trpc.stripe.getBillingInfo.pathFilter()),
        ]);
        router.refresh();
        toast.info("Subscription resumed");
      },
      onError: (err) => {
        toast.error(
          "Failed to resume subscription. Please try again or contact support@unkey.dev.",
        );
        console.error("Subscription resumption error:", err);
      },
    }),
  );

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
          loading={uncancelSubscription.isPending}
          disabled={uncancelSubscription.isPending}
          onClick={() => uncancelSubscription.mutate()}
        >
          Resubscribe
        </Button>
      </div>
    </SettingCard>
  );
};
