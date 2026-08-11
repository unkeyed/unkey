"use client";

import { trpc } from "@/lib/trpc/client";
import { TriangleWarning2 } from "@unkey/icons";
import { Button, DialogContainer, InfoTooltip, ItemFooter, ItemSeparator, toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { ADMIN_ONLY_TOOLTIP } from "./constants";

type CancelLinksProps = {
  isAdmin: boolean;
  /** True when the API subscription is active and not already ending. */
  canCancelApi: boolean;
};

/**
 * Cancelling is a quiet footer on the Plans card rather than a danger zone: it is
 * a rare action that belongs beside the plans it ends, not a section of its own.
 * The two products cancel differently — Compute stops immediately, the API plan
 * runs to the end of the period — so each keeps its own confirmation.
 */
export function CancelLinks({ isAdmin, canCancelApi }: CancelLinksProps) {
  const [openDialog, setOpenDialog] = useState<"compute" | "api" | null>(null);

  const { data: subscription } = trpc.stripe.getDeploySubscription.useQuery(undefined, {
    staleTime: 30_000,
  });
  const hasComputePlan = Boolean(subscription?.plan);

  if (!hasComputePlan && !canCancelApi) {
    return null;
  }

  return (
    <>
      <ItemSeparator />
      <ItemFooter className="justify-end gap-4">
        {hasComputePlan ? (
          <CancelLink isAdmin={isAdmin} onClick={() => setOpenDialog("compute")}>
            Cancel Compute
          </CancelLink>
        ) : null}
        {canCancelApi ? (
          <CancelLink isAdmin={isAdmin} onClick={() => setOpenDialog("api")}>
            Cancel API plan
          </CancelLink>
        ) : null}
      </ItemFooter>

      <CancelComputeDialog
        open={openDialog === "compute"}
        onOpenChange={(open) => setOpenDialog(open ? "compute" : null)}
      />
      <CancelApiDialog
        open={openDialog === "api"}
        onOpenChange={(open) => setOpenDialog(open ? "api" : null)}
      />
    </>
  );
}

function CancelLink({
  isAdmin,
  onClick,
  children,
}: {
  isAdmin: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <InfoTooltip content={ADMIN_ONLY_TOOLTIP} disabled={isAdmin} asChild>
      <span>
        <button
          type="button"
          className="text-[13px] text-gray-9 transition-colors hover:text-gray-11 disabled:cursor-not-allowed"
          disabled={!isAdmin}
          onClick={onClick}
        >
          {children}
        </button>
      </span>
    </InfoTooltip>
  );
}

function CancelComputeDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const trpcUtils = trpc.useUtils();

  const cancel = trpc.stripe.cancelDeploy.useMutation({
    onSuccess: async () => {
      onOpenChange(false);
      toast.info("Compute cancelled");
      await Promise.all([
        trpcUtils.stripe.getDeploySubscription.invalidate(),
        trpcUtils.stripe.getDeployEntitlement.invalidate(),
        trpcUtils.stripe.getUpcomingInvoice.invalidate(),
        trpcUtils.billing.queryDeployUsage.invalidate(),
        trpcUtils.workspace.getCurrent.invalidate(),
        trpcUtils.stripe.getDeploySubscription.refetch(),
      ]);
    },
    onError: (err) => toast.error(err.message),
  });

  return (
    <DialogContainer
      isOpen={open}
      onOpenChange={onOpenChange}
      title="Cancel Compute"
      subTitle="Turn off Compute for this workspace"
      footer={
        <Button
          type="button"
          variant="primary"
          color="danger"
          size="xlg"
          className="w-full rounded-lg"
          loading={cancel.isLoading}
          onClick={() => cancel.mutate()}
        >
          Cancel Compute
        </Button>
      }
    >
      <div className="text-[13px] text-gray-11 leading-6">
        Cancelling stops Compute immediately: your deployments stop and no further usage is billed.
        Usage up to now is still charged, and the plan fee already paid is not refunded.
      </div>
    </DialogContainer>
  );
}

function CancelApiDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const router = useRouter();
  const trpcUtils = trpc.useUtils();

  const cancel = trpc.stripe.cancelSubscription.useMutation({
    onSuccess: async () => {
      await Promise.all([
        trpcUtils.workspace.getCurrent.invalidate(),
        trpcUtils.billing.queryUsage.invalidate(),
        trpcUtils.stripe.getBillingInfo.invalidate(),
        trpcUtils.stripe.getUpcomingInvoice.invalidate(),
      ]);
      router.refresh();
      onOpenChange(false);
      toast.info("Subscription cancelled");
    },
    onError: (err) => toast.error(err.message),
  });

  return (
    <DialogContainer
      isOpen={open}
      onOpenChange={onOpenChange}
      title="Cancel API plan"
      subTitle="Downgrade your workspace to the free tier"
      footer={
        <div className="flex w-full flex-col items-center justify-center gap-2">
          <Button
            type="button"
            variant="primary"
            color="danger"
            size="xlg"
            className="w-full rounded-lg"
            loading={cancel.isLoading}
            onClick={() => cancel.mutate()}
          >
            Cancel API plan
          </Button>
          <div className="text-gray-9 text-xs">
            You can resume your subscription until the end of the billing period
          </div>
        </div>
      }
    >
      <div className="flex items-center gap-4 rounded-xl border border-errorA-3 bg-errorA-2 px-[22px] py-6 dark:bg-black">
        <div className="flex size-8 shrink-0 items-center justify-center rounded-full bg-error-9">
          <TriangleWarning2 iconSize="sm-regular" className="text-white" />
        </div>
        <div className="text-[13px] text-error-12 leading-6">
          <span className="font-medium">Warning:</span> cancelling your API plan will downgrade your
          workspace to the free tier at the end of the current billing period. You will lose access
          to paid features, usage limits will be reduced, and all team members other than you will
          be deactivated.
        </div>
      </div>
    </DialogContainer>
  );
}
