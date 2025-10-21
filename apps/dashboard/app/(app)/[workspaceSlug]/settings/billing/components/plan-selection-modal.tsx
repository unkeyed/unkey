"use client";

import { formatNumber } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { Button, DialogContainer, toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

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
  const [selectedProductId, setSelectedProductId] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [hasMounted, setHasMounted] = useState(false);
  const router = useRouter();
  const trpcUtils = trpc.useUtils();

  // Initialize selectedProductId after mount to prevent hydration mismatch
  useEffect(() => {
    setHasMounted(true);
    if (currentProductId) {
      setSelectedProductId(currentProductId);
    }
  }, [currentProductId]);

  const handleOpenChange = useCallback(
    (open: boolean) => {
      onOpenChange?.(open);
    },
    [onOpenChange],
  );

  const createSubscription = trpc.stripe.createSubscription.useMutation({
    onSuccess: () => {
      setIsLoading(false);
      handleOpenChange(false);
      toast.success("Plan activated successfully!");
      trpcUtils.stripe.getBillingInfo.invalidate();
      trpcUtils.billing.queryUsage.invalidate();
      router.refresh();
      router.push(`/${workspaceSlug}/settings/billing`);
    },
    onError: (err) => {
      setIsLoading(false);
      toast.error(err.message);
    },
  });

  const updateSubscription = trpc.stripe.updateSubscription.useMutation({
    onSuccess: async () => {
      setIsLoading(false);

      toast.success("Plan changed successfully!");
      await trpcUtils.stripe.getBillingInfo.invalidate();
      await trpcUtils.billing.queryUsage.invalidate();
      await trpcUtils.workspace.getCurrent.invalidate();
      // await trpcUtils.workspace.getCurrent.refetch();
      handleOpenChange(false);
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

    if (selectedProductId === "free" && !isChangingPlan) {
      // For free tier on initial selection, just close modal and redirect
      setIsLoading(false);
      handleOpenChange(false);
      toast.success("Staying on Free plan - you can upgrade anytime!");
      router.push(`/${workspaceSlug}/settings/billing`);
      return;
    }

    if (isChangingPlan && currentProductId) {
      // Update existing subscription
      await updateSubscription.mutateAsync({
        oldProductId: currentProductId,
        newProductId: selectedProductId,
      });
    } else {
      // Create new subscription
      await createSubscription.mutateAsync({
        productId: selectedProductId,
      });
    }
  };

  const handleSkip = () => {
    handleOpenChange(false);
    if (!isChangingPlan) {
      toast.info("Payment method added - you can upgrade anytime from billing settings!");
      router.push(`/${workspaceSlug}/settings/billing`);
    }
  };

  const selectedProduct = products.find((p) => p.id === selectedProductId);

  const handleCloseAttempt = () => {
    toast.info(
      isChangingPlan
        ? "Please select a plan or click 'Cancel' to close"
        : "Please select a plan or choose 'I'll choose later' to continue",
    );
  };

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
        showCloseWarning={true}
        onAttemptClose={() => {}}
        footer={<div className="w-full flex flex-col gap-3" />}
      >
        <div className="space-y-4">
          <div className="animate-pulse">
            <div className="h-16 bg-gray-200 dark:bg-gray-700 rounded-lg mb-2" />
            <div className="h-16 bg-gray-200 dark:bg-gray-700 rounded-lg mb-2" />
            <div className="h-16 bg-gray-200 dark:bg-gray-700 rounded-lg mb-2" />
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
      showCloseWarning={true}
      onAttemptClose={handleCloseAttempt}
      footer={
        <div className="w-full flex flex-col gap-3">
          <Button
            type="button"
            variant="primary"
            size="lg"
            loading={isLoading}
            disabled={!selectedProductId || isLoading}
            className="w-full"
            onClick={handleSelectPlan}
          >
            {selectedProductId === "free" && !isChangingPlan
              ? "Continue with Free Plan"
              : selectedProduct
                ? `${isChangingPlan ? "Change to" : "Start"} ${
                    selectedProduct.name
                  } Plan - $${selectedProduct.dollar}/mo`
                : "Select a plan first"}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={isLoading}
            className="w-full text-gray-11 hover:text-gray-12"
            onClick={handleSkip}
          >
            {isChangingPlan ? "Cancel" : "Skip for now - Stay on Free Plan"}
          </Button>
        </div>
      }
    >
      <div className="space-y-4">
        <div className="space-y-2">
          {/* Paid Plans */}
          {products.map((product) => (
            <label
              key={product.id}
              className={cn(
                "border rounded-lg px-4 py-1 cursor-pointer transition-all hover:border-gray-6 bg-white dark:bg-black block",
                selectedProductId === product.id
                  ? "border-info-7 bg-info-2 ring-1 ring-info-7"
                  : currentProductId === product.id
                    ? "border-gray-5 bg-gray-2"
                    : "border-gray-4",
              )}
            >
              <input
                type="radio"
                name="product-selection"
                value={product.id}
                checked={selectedProductId === product.id}
                onChange={() => setSelectedProductId(product.id)}
                className="sr-only"
              />
              <div className="flex items-center justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-3">
                    <div
                      className={cn(
                        "w-4 h-4 rounded-full border-2 flex items-center justify-center",
                        selectedProductId === product.id
                          ? "border-info-9 bg-info-9"
                          : "border-gray-6",
                      )}
                    >
                      {selectedProductId === product.id && (
                        <div className="w-2 h-2 bg-white rounded-full" />
                      )}
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <h3 className="font-semibold text-gray-12">{product.name}</h3>
                        {currentProductId === product.id && (
                          <span className="text-xs bg-info-3 text-info-11 px-2 py-0.5 rounded-full">
                            Current
                          </span>
                        )}
                      </div>
                      <p className="text-sm text-gray-11">
                        {formatNumber(product.quotas.requestsPerMonth)} requests/month
                      </p>
                    </div>
                  </div>
                </div>
                <div className="text-right">
                  <div className="font-semibold text-gray-12">
                    ${product.dollar}
                    <span className="text-sm font-normal text-gray-11">/mo</span>
                  </div>
                </div>
              </div>
            </label>
          ))}
        </div>

        <div className="text-center">
          <p className="text-xs text-gray-9">
            You can change or cancel your plan anytime from the billing settings.
          </p>
        </div>
      </div>
    </DialogContainer>
  );
};
