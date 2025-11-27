"use client";
import { useTRPC } from "@/lib/trpc/client";
import { Button, SettingCard, toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { Confirm } from "./confirmation";

import { useMutation } from "@tanstack/react-query";
import { useQueryClient } from "@tanstack/react-query";

export const CancelPlan: React.FC = () => {
  const trpc = useTRPC();
  const queryClient = useQueryClient();
  const router = useRouter();

  const cancelSubscription = useMutation(
    trpc.stripe.cancelSubscription.mutationOptions({
      onSuccess: async () => {
        // Revalidate helper: invalidate AND explicitly refetch to ensure UI updates
        await Promise.all([
          queryClient.invalidateQueries(trpc.workspace.getCurrent.pathFilter()),
          queryClient.invalidateQueries(trpc.billing.queryUsage.pathFilter()),
          queryClient.invalidateQueries(trpc.stripe.getBillingInfo.pathFilter()),
          queryClient.refetchQueries(trpc.workspace.getCurrent.pathFilter()),
          queryClient.refetchQueries(trpc.billing.queryUsage.pathFilter()),
          queryClient.refetchQueries(trpc.stripe.getBillingInfo.pathFilter()),
        ]);
        router.refresh();
        toast.info("Subscription cancelled");
      },
      onError: (err) => {
        toast.error(err.message);
      },
    }),
  );

  return (
    <SettingCard
      title="Cancel Subscription"
      description="Cancelling your subscription will downgrade your workspace to the free tier."
      border="both"
      className="border-t w-full"
      contentWidth="w-full lg:w-[320px]"
    >
      <div className="w-full flex h-full items-center justify-end gap-4">
        <Confirm
          title="Cancel plan"
          description="Canceling your plan will downgrade your workspace to the free tier at the end of the current period. You can resume your subscription until then."
          onConfirm={() => cancelSubscription.mutateAsync()}
          trigger={(onClick) => (
            <Button variant="outline" color="danger" size="lg" onClick={onClick}>
              Cancel Plan
            </Button>
          )}
        />
      </div>
    </SettingCard>
  );
};
