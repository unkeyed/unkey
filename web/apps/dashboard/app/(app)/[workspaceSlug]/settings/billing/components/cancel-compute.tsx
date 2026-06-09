"use client";

import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { Button, DialogContainer, toast } from "@unkey/ui";
import { useState } from "react";
import { BillingCard, billingButton } from "./billing-card";

type CancelComputeProps = {
  disabled?: boolean;
  disabledReason?: string;
};

/**
 * Danger-zone row for cancelling the Compute plan. Self-contained (row,
 * confirm dialog, mutation) so it can sit in the shared danger zone next to
 * the workspace-plan cancellation.
 */
export const CancelCompute: React.FC<CancelComputeProps> = ({
  disabled = false,
  disabledReason,
}) => {
  const trpcUtils = trpc.useUtils();
  const [isDialogOpen, setDialogOpen] = useState(false);

  const cancel = trpc.stripe.cancelDeploy.useMutation({
    onSuccess: async () => {
      setDialogOpen(false);
      toast.info("Compute cancelled");
      await Promise.all([
        trpcUtils.stripe.getDeploySubscription.invalidate(),
        trpcUtils.workspace.getCurrent.invalidate(),
        trpcUtils.stripe.getDeploySubscription.refetch(),
      ]);
    },
    onError: (err) => toast.error(err.message),
  });

  return (
    <>
      <BillingCard
        title="Cancel Compute subscription"
        description={
          disabled && disabledReason
            ? disabledReason
            : "Stops Compute immediately. Usage so far is billed; the plan fee is not refunded. Your API plan is not affected."
        }
      >
        <Button
          variant="outline"
          color="danger"
          size="lg"
          className={billingButton}
          disabled={disabled}
          onClick={() => setDialogOpen(true)}
        >
          Cancel Compute
        </Button>
      </BillingCard>

      <DialogContainer
        isOpen={isDialogOpen}
        onOpenChange={setDialogOpen}
        title="Cancel Compute"
        subTitle="Turn off Compute for this workspace"
        className="rounded-none!"
        footer={
          <Button
            type="button"
            variant="primary"
            color="danger"
            size="xlg"
            className={cn("w-full", billingButton)}
            loading={cancel.isLoading}
            onClick={() => cancel.mutate()}
          >
            Cancel Compute
          </Button>
        }
      >
        <div className="text-gray-11 text-[13px] leading-6">
          Cancelling stops Compute immediately: your deployments stop and no further usage is
          billed. Usage up to now is still charged, and the plan fee already paid is not refunded.
        </div>
      </DialogContainer>
    </>
  );
};
