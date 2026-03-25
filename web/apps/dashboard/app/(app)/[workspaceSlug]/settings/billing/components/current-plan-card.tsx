"use client";
import { Button, SettingCard } from "@unkey/ui";
import { useCallback } from "react";

type CurrentPlanCardProps = {
  currentProduct?: {
    name: string;
    dollar: number;
    quotas?: { requestsPerMonth: number };
  };
  onChangePlan?: () => void;
};

export const CurrentPlanCard = ({ currentProduct, onChangePlan }: CurrentPlanCardProps) => {
  const handleChangePlan = useCallback(() => {
    onChangePlan?.();
  }, [onChangePlan]);

  const planName = currentProduct?.name ?? "Free Plan";
  const price = currentProduct?.dollar ?? 0;

  return (
    <SettingCard
      title={
        <div className="flex items-center gap-2">
          <span>{planName}</span>
          <span className="text-xs bg-info-3 text-info-11 px-2 py-0.5 rounded-full font-normal">
            Active
          </span>
        </div>
      }
      description={`$${price}/mo`}
      contentWidth="w-full lg:w-[320px]"
    >
      <div className="w-full flex h-full items-center justify-end gap-4">
        <Button
          variant="outline"
          className="px-2.5 py-3 text-gray-12 font-medium text-sm bg-grayA-2 hover:bg-grayA-3"
          onClick={handleChangePlan}
        >
          {currentProduct ? "Change Plan" : "Upgrade"}
        </Button>
      </div>
    </SettingCard>
  );
};
