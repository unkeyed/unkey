"use client";
import { Button, InfoTooltip } from "@unkey/ui";
import { useCallback } from "react";
import { BillingCard, billingButton } from "./billing-card";

type CurrentPlanCardProps = {
  currentProduct?: {
    name: string;
    dollar: number;
    quotas?: { requestsPerMonth: number };
  };
  onChangePlan?: () => void;
  disabled?: boolean;
  disabledReason?: string;
};

export const CurrentPlanCard = ({
  currentProduct,
  onChangePlan,
  disabled = false,
  disabledReason,
}: CurrentPlanCardProps) => {
  const handleChangePlan = useCallback(() => {
    onChangePlan?.();
  }, [onChangePlan]);

  const planName = currentProduct?.name ?? "Free Plan";
  const price = currentProduct?.dollar ?? 0;

  return (
    <BillingCard
      label="Current plan"
      title={
        <div className="flex items-center gap-2.5">
          <span className="font-medium text-base text-gray-12 tracking-tight">{planName}</span>
          <span className="rounded-sm border border-successA-3 bg-successA-2 px-1.5 py-0.5 font-mono text-[10px] text-success-11 uppercase tracking-wider">
            Active
          </span>
        </div>
      }
      description={<span className="font-mono text-[13px] text-gray-10">${price}/mo</span>}
    >
      <InfoTooltip content={disabledReason ?? ""} disabled={!disabled || !disabledReason} asChild>
        <span>
          <Button
            variant="outline"
            size="lg"
            className={billingButton}
            onClick={handleChangePlan}
            disabled={disabled}
          >
            {currentProduct ? "Change Plan" : "Upgrade"}
          </Button>
        </span>
      </InfoTooltip>
    </BillingCard>
  );
};
