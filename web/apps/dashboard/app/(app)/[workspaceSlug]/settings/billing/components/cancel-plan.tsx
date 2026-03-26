"use client";
import { trpc } from "@/lib/trpc/client";
import { TriangleWarning2 } from "@unkey/icons";
import { Button, DialogContainer, SettingsZoneRow, toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useState } from "react";

export const CancelPlan: React.FC = () => {
  const trpcUtils = trpc.useUtils();
  const router = useRouter();
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  const cancelSubscription = trpc.stripe.cancelSubscription.useMutation({
    onSuccess: async () => {
      // Revalidate helper: invalidate AND explicitly refetch to ensure UI updates
      await Promise.all([
        trpcUtils.workspace.getCurrent.invalidate(),
        trpcUtils.billing.queryUsage.invalidate(),
        trpcUtils.stripe.getBillingInfo.invalidate(),
        trpcUtils.workspace.getCurrent.refetch(),
        trpcUtils.billing.queryUsage.refetch(),
        trpcUtils.stripe.getBillingInfo.refetch(),
      ]);
      router.refresh();
      setIsDialogOpen(false);
      toast.info("Subscription cancelled");
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });

  return (
    <>
      <SettingsZoneRow
        title="Cancel subscription"
        description="Cancelling your subscription will downgrade your workspace to the free tier."
        action={{
          label: "Cancel Plan",
          onClick: () => setIsDialogOpen(true),
        }}
      />

      <DialogContainer
        isOpen={isDialogOpen}
        onOpenChange={setIsDialogOpen}
        title="Cancel subscription"
        subTitle="Downgrade your workspace to the free tier"
        footer={
          <div className="w-full flex flex-col gap-2 items-center justify-center">
            <Button
              type="button"
              variant="primary"
              color="danger"
              size="xlg"
              className="w-full rounded-lg"
              loading={cancelSubscription.isLoading}
              onClick={() => cancelSubscription.mutate()}
            >
              Cancel subscription
            </Button>
            <div className="text-gray-9 text-xs">
              You can resume your subscription until the end of the billing period
            </div>
          </div>
        }
      >
        <div className="rounded-xl bg-errorA-2 dark:bg-black border border-errorA-3 flex items-center gap-4 px-[22px] py-6">
          <div className="bg-error-9 size-8 rounded-full flex items-center justify-center shrink-0">
            <TriangleWarning2 iconSize="sm-regular" className="text-white" />
          </div>
          <div className="text-error-12 text-[13px] leading-6">
            <span className="font-medium">Warning:</span> cancelling your subscription will
            downgrade your workspace to the free tier at the end of the current billing period. You
            will lose access to paid features and usage limits will be reduced.
          </div>
        </div>
      </DialogContainer>
    </>
  );
};
