"use client";
import { formatNumber } from "@/lib/fmt";
import { Button, SettingCard } from "@unkey/ui";
import { useCallback } from "react";

type CurrentPlanCardProps = {
  currentProduct?: {
    name: string;
    dollar: number;
    quotas: { requestsPerMonth: number };
  };
  onChangePlan?: () => void;
};

export const CurrentPlanCard = ({ currentProduct, onChangePlan }: CurrentPlanCardProps) => {
  const handleChangePlan = useCallback(() => {
    onChangePlan?.();
  }, [onChangePlan]);
  return (
    <SettingCard
      title="Current Plan"
      description={<div className="min-w-[300px]">Your active subscription plan</div>}
      border="both"
      className="w-full min-w-[200px]"
      contentWidth="w-full"
    >
      <div className="w-full flex h-full items-center justify-end gap-4">
        <div className="flex-1 shrink">
          {currentProduct ? (
            <div className="space-y-1">
              <div className="flex items-center gap-2">
                <h3 className="font-semibold text-gray-12">{currentProduct.name}</h3>
                <span className="text-xs bg-info-3 text-info-11 px-2 py-0.5 rounded-full">
                  Active
                </span>
              </div>
              <div className="flex items-center gap-4 text-sm text-gray-11">
                <span>{formatNumber(currentProduct.quotas.requestsPerMonth)} requests/month</span>
                <span className="font-medium text-gray-12">${currentProduct.dollar}/mo</span>
              </div>
            </div>
          ) : (
            <div className="space-y-1">
              <div className="flex items-center gap-2">
                <h3 className="font-semibold text-gray-12">Free Plan</h3>
                <span className="text-xs bg-gray-3 text-gray-11 px-2 py-0.5 rounded-full">
                  Active
                </span>
              </div>
              <div className="flex items-center gap-4 text-sm text-gray-11">
                <span>150,000 requests/month</span>
                <span className="font-medium text-gray-12">$0/mo</span>
              </div>
            </div>
          )}
        </div>
        <Button
          size="lg"
          variant={currentProduct ? "outline" : "primary"}
          onClick={handleChangePlan}
        >
          {currentProduct ? "Change Plan" : "Upgrade"}
        </Button>
      </div>
    </SettingCard>
  );
};
