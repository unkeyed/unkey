"use client";
import { trpc } from "@/lib/trpc/client";
import { Button, SettingCard, toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { Confirm } from "./confirmation";

export const CancelPlan: React.FC = () => {
  const trpcUtils = trpc.useUtils();
  const router = useRouter();

  const cancelSubscription = trpc.stripe.cancelSubscription.useMutation({
    onSuccess: () => {
      trpcUtils.workspace.getCurrent.invalidate();
      trpcUtils.billing.queryUsage.invalidate();
      trpcUtils.stripe.getBillingInfo.invalidate();
      router.refresh();
      toast.info("Subscription cancelled");
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });

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
