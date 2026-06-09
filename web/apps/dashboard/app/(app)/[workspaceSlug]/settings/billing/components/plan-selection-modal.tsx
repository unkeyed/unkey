"use client";

import { formatNumber } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { Button, DialogContainer, toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { billingButton } from "./billing-card";

type PlanSelectionModalProps = {
  isOpen: boolean;
  onOpenChange?: (open: boolean) => void;
  products: Array<{
    id: string;
    name: string;
    priceId: string;
    dollar: number;
    quotas: {
      requestsPerMonth: number;
    };
  }>;
  workspaceSlug: string;
  currentProductId?: string;
  isChangingPlan?: boolean;
};

export const PlanSelectionModal = ({
  isOpen,
  onOpenChange,
  products,
  workspaceSlug,
  currentProductId,
  isChangingPlan = false,
}: PlanSelectionModalProps) => {
  const [selectedProductId, setSelectedProductId] = useState<string | null>(
    currentProductId ?? null,
  );
  const [isLoading, setIsLoading] = useState(false);
  const [hasMounted, setHasMounted] = useState(false);
  const router = useRouter();
  const trpcUtils = trpc.useUtils();

  // Set hasMounted flag after initial mount to prevent hydration mismatch
  useEffect(() => {
    setHasMounted(true);
  }, []);

  const handleOpenChange = useCallback(
    (open: boolean) => {
      onOpenChange?.(open);
    },
    [onOpenChange],
  );

  const revalidateData = useCallback(async () => {
    await Promise.all([
      trpcUtils.stripe.getBillingInfo.invalidate(),
      trpcUtils.billing.queryUsage.invalidate(),
      trpcUtils.workspace.getCurrent.invalidate(),
      trpcUtils.workspace.getCurrent.refetch(),
    ]);
  }, [trpcUtils]);

  const createSubscription = trpc.stripe.createSubscription.useMutation({
    onSuccess: async () => {
      handleOpenChange(false);
      setIsLoading(false);
      toast.success("Plan activated successfully!");
      await revalidateData();
      router.push(`/${workspaceSlug}/settings/billing`);
    },
    onError: (err) => {
      setIsLoading(false);
      toast.error(err.message);
    },
  });

  const updateSubscription = trpc.stripe.updateSubscription.useMutation({
    onSuccess: async () => {
      handleOpenChange(false);
      setIsLoading(false);
      toast.success("Plan changed successfully!");
      await revalidateData();
    },
    onError: (err) => {
      setIsLoading(false);
      toast.error(err.message);
    },
  });

  const handleSelectPlan = async () => {
    if (!selectedProductId) {
      return;
    }

    setIsLoading(true);
    if (isChangingPlan && currentProductId) {
      // Update existing subscription. The current product is derived from
      // the existing subscription server-side, so we no longer need to send
      // `oldProductId` from the client.
      await updateSubscription.mutateAsync({
        newProductId: selectedProductId,
      });
    } else {
      // Create new subscription
      await createSubscription.mutateAsync({
        productId: selectedProductId,
      });
    }
  };

  const handleSkip = async () => {
    if (!isChangingPlan) {
      await revalidateData();
      // Wait for workspace data to be refetched before navigation
      toast.info("Payment method added - you can upgrade anytime from billing settings!");
      router.push(`/${workspaceSlug}/settings/billing`);
    }
    handleOpenChange(false);
  };

  const selectedProduct = products.find((p) => p.id === selectedProductId);

  // Don't render modal content until after hydration
  if (!hasMounted) {
    return (
      <DialogContainer
        isOpen={isOpen}
        onOpenChange={(open) => onOpenChange?.(open)}
        title={isChangingPlan ? "Change Your Plan" : "Choose Your Plan"}
        subTitle={
          isChangingPlan
            ? "Select a new plan to switch to"
            : "Select a plan to get started with your new payment method"
        }
        className="rounded-none!"
        showCloseWarning={true}
        onAttemptClose={() => {}}
        footer={<div className="w-full flex flex-col gap-3" />}
      >
        <div className="flex flex-col gap-4">
          <div className="animate-pulse grid grid-cols-3 gap-px">
            <div className="h-28 bg-grayA-3" />
            <div className="h-28 bg-grayA-3" />
            <div className="h-28 bg-grayA-3" />
          </div>
        </div>
      </DialogContainer>
    );
  }

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={handleOpenChange}
      title={isChangingPlan ? "Change Your Plan" : "Choose Your Plan"}
      subTitle={
        isChangingPlan
          ? "Select a new plan to switch to"
          : "Select a plan to get started with your new payment method"
      }
      className="rounded-none!"
      showCloseWarning={true}
      onAttemptClose={handleSkip}
      footer={
        <div className="w-full flex flex-col gap-6">
          <Button
            type="button"
            variant="primary"
            size="lg"
            loading={isLoading}
            disabled={!selectedProductId || isLoading}
            className={cn("w-full", billingButton)}
            onClick={handleSelectPlan}
          >
            {selectedProduct
              ? `${isChangingPlan ? "Change to" : "Start"} ${
                  selectedProduct.name
                } Plan - $${selectedProduct.dollar}/mo`
              : "Select a plan first"}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="lg"
            disabled={isLoading}
            className={cn("w-full text-gray-11 hover:text-gray-12", billingButton)}
            onClick={handleSkip}
          >
            {isChangingPlan ? "Cancel" : "Skip for now - Stay on Free Plan"}
          </Button>
        </div>
      }
    >
      <div className="flex flex-col gap-4">
        <div className="grid grid-cols-1 border-t border-l border-grayA-4 sm:grid-cols-2 lg:grid-cols-3">
          {products.map((product) => {
            const isSelected = selectedProductId === product.id;
            const isCurrent = currentProductId === product.id;
            return (
              <label
                key={product.id}
                className={cn(
                  "relative flex min-w-0 cursor-pointer flex-col gap-5 border-r border-b border-grayA-4 px-5 pt-5 pb-6 transition-colors",
                  "focus-within:bg-grayA-2",
                  isSelected ? "bg-grayA-2 ring-1 ring-inset ring-gray-12" : "hover:bg-grayA-2",
                )}
              >
                <input
                  type="radio"
                  name="product-selection"
                  value={product.id}
                  checked={isSelected}
                  onChange={() => setSelectedProductId(product.id)}
                  className="sr-only"
                />
                <div className="flex items-center justify-between">
                  <span className="font-mono text-[11px] text-gray-11 uppercase tracking-[0.08em]">
                    {product.name}
                  </span>
                  {isCurrent ? (
                    <span className="font-mono text-[10px] text-gray-9 uppercase tracking-wider">
                      Current
                    </span>
                  ) : null}
                </div>
                <div className="flex items-baseline gap-1">
                  <span className="font-medium text-3xl text-gray-12 tracking-tight">
                    ${product.dollar}
                  </span>
                  <span className="text-gray-9 text-sm">/mo</span>
                </div>
                <p className="text-[13px] text-gray-10 leading-snug">
                  {formatNumber(product.quotas.requestsPerMonth)} requests/month
                </p>
              </label>
            );
          })}
        </div>

        <div className="text-center">
          <p className="text-gray-9 text-xs">
            You can change or cancel your plan anytime from the billing settings.
          </p>
        </div>
      </div>
    </DialogContainer>
  );
};
