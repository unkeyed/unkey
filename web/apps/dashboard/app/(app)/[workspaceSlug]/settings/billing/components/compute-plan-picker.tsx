"use client";

import { formatDollars } from "@/lib/fmt";
import type { DeployPlan } from "@/lib/stripe/deployPlan";
import type { DeployPlanOption } from "@/lib/trpc/routers/stripe/getDeployPlans";
import { cn } from "@/lib/utils";
import { ArrowRight, ArrowUpRight, Check, CircleInfo } from "@unkey/icons";
import { Button, DialogContainer } from "@unkey/ui";
import {
  ALL_PLANS_INCLUDE,
  COMPUTE_PLANS_LINK_HREF,
  CREDITS_INFO,
  CREDITS_LINK_HREF,
  CREDITS_LINK_LABEL,
  FEATURES,
  PLAN_BLURBS,
} from "./compute-plan-copy";
import { PlanTierIcon } from "./plan-tier-icons";

// Deliberately separate from PlanChangeModal (the API add-on picker): Compute
// uses stacked rows with a per-row commit button, not radio rows with a
// single bottom CTA.

type ComputePlanRowsProps = {
  plans: DeployPlanOption[];
  currentPlan?: DeployPlan | null;
  submittingPlan?: DeployPlan | null;
  onSelect: (plan: DeployPlan) => void;
  selectLabel?: (plan: DeployPlanOption) => string;
  warningFor?: (plan: DeployPlanOption) => string | null;
  disabledReason?: string;
};

type RowProps = ComputePlanRowsProps & { plan: DeployPlanOption };

function intervalSuffix(interval: string | null): string {
  switch (interval) {
    case "month":
      return "/mo";
    case "year":
      return "/yr";
    default:
      return interval ? `/${interval}` : "";
  }
}

function Row({
  plan,
  currentPlan,
  submittingPlan,
  onSelect,
  selectLabel,
  warningFor,
  disabledReason,
}: RowProps) {
  const isCurrent = plan.plan === currentPlan;
  const isSubmitting = plan.plan === submittingPlan;
  const disabled = isCurrent || isSubmitting || disabledReason !== undefined;
  const blurb = PLAN_BLURBS[plan.plan] ?? plan.description ?? "";
  const warning = !isCurrent && warningFor ? warningFor(plan) : null;
  const label = selectLabel ? selectLabel(plan) : "Select";

  const cardClassName =
    "group flex w-full items-center gap-[13px] rounded-[11px] border border-gray-4 bg-gray-1 px-[15px] py-3 text-left transition-colors";

  const details = (
    <>
      <span className="flex size-9 shrink-0 items-center justify-center rounded-[9px] bg-grayA-3 text-gray-12">
        <PlanTierIcon plan={plan.plan} className="size-[19px]" />
      </span>

      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="font-medium text-[15px] text-gray-12">{plan.name}</span>
        {blurb ? <span className="text-[13px] text-gray-11">{blurb}</span> : null}
      </span>

      <span className="shrink-0 text-right">
        {plan.amount !== null ? (
          <>
            <span className="font-semibold text-[15px] text-gray-12 tabular-nums">
              {formatDollars(plan.amount)}
            </span>
            <span className="text-[12px] text-gray-11">{intervalSuffix(plan.interval)}</span>
          </>
        ) : (
          <span className="font-semibold text-[15px] text-gray-12">Contact us</span>
        )}
      </span>
    </>
  );

  return (
    <div className="flex flex-col gap-2">
      {isCurrent ? (
        <div className={cardClassName}>
          {details}
          <span className="shrink-0 rounded-md bg-info-3 px-2.5 py-1.5 text-[13px] text-info-11 leading-none">
            Current
          </span>
        </div>
      ) : (
        <button
          type="button"
          disabled={disabled}
          aria-busy={isSubmitting}
          title={disabledReason}
          onClick={() => onSelect(plan.plan)}
          className={cn(
            cardClassName,
            disabled ? "cursor-not-allowed opacity-70" : "cursor-pointer hover:border-gray-6",
          )}
        >
          {details}
          <span className="inline-flex shrink-0 items-center gap-1.5 rounded-lg bg-gray-12 px-3 py-2 text-[13px] font-medium text-gray-1">
            {isSubmitting ? (
              <>
                <span
                  className="size-4 animate-spin rounded-full border-2 border-gray-1 border-t-transparent"
                  aria-hidden="true"
                />
                <span className="sr-only">Working…</span>
              </>
            ) : (
              <>
                {label}
                <ArrowRight
                  iconSize="md-regular"
                  className="transition-transform group-hover:translate-x-0.5"
                />
              </>
            )}
          </span>
        </button>
      )}

      {warning ? <p className="text-[13px] text-warning-11 leading-5">{warning}</p> : null}
    </div>
  );
}

export function ComputePlanRows(props: ComputePlanRowsProps) {
  return (
    <div className="flex flex-col gap-2.5 pt-2">
      {props.plans.map((plan) => (
        <Row key={plan.plan} {...props} plan={plan} />
      ))}
    </div>
  );
}

export function ComputePlanFeatures() {
  return (
    <div className="grid grid-cols-2 gap-x-6 gap-y-6 py-2">
      {FEATURES.map(({ Icon, title, description }) => (
        <div key={title}>
          <div className="flex items-center gap-[9px]">
            <Icon iconSize="lg-regular" className="shrink-0 text-gray-12" />
            <span className="font-medium text-[13px] text-gray-12">{title}</span>
          </div>
          <p className="mt-1 text-[12.5px] text-gray-11 leading-relaxed">{description}</p>
        </div>
      ))}
    </div>
  );
}

export function ComputePlansMoreInfo() {
  return (
    <p className="text-[13px] text-gray-11 leading-normal">
      Get more information about {/* @dh todo - add docs */}
      <a
        href={COMPUTE_PLANS_LINK_HREF}
        target="_blank"
        rel="noopener noreferrer"
        className="font-medium text-info-11 hover:underline"
      >
        Compute plans
      </a>
      .
    </p>
  );
}

export function AllPlansInclude() {
  return (
    <div className="rounded-[11px] border border-gray-4 bg-gray-1 px-4 py-3.5">
      <span className="font-medium text-[13px] text-gray-12">Included in every plan</span>
      <ul className="mt-3 grid grid-cols-2 gap-x-5 gap-y-2.5">
        {ALL_PLANS_INCLUDE.map((feature) => (
          <li key={feature} className="flex items-center gap-2.5 text-[13px] text-gray-11">
            <Check iconSize="md-regular" className="shrink-0 text-gray-10" />
            {feature}
          </li>
        ))}
      </ul>
    </div>
  );
}

export function CreditsInfoStrip() {
  return (
    <div className="flex items-start gap-2.5 rounded-[11px] border border-gray-4 bg-gray-1 px-3.5 py-3">
      <CircleInfo iconSize="lg-regular" className="mt-px shrink-0 text-info-9" />
      <p className="text-[12.5px] text-gray-11 leading-relaxed">
        {CREDITS_INFO} {/* @dh todo - add docs */}
        <a
          href={CREDITS_LINK_HREF}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-0.5 whitespace-nowrap font-medium text-info-11 hover:underline"
        >
          {CREDITS_LINK_LABEL}
          <ArrowUpRight iconSize="sm-regular" className="size-3" />
        </a>
      </p>
    </div>
  );
}

type ComputePlanConfirmDialogProps = {
  /** Null keeps the dialog closed. */
  plan: DeployPlanOption | null;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
  isLoading: boolean;
  currentPlanName?: string;
  note?: string;
};

export function ComputePlanConfirmDialog({
  plan,
  onOpenChange,
  onConfirm,
  isLoading,
  currentPlanName,
  note,
}: ComputePlanConfirmDialogProps) {
  return (
    <DialogContainer
      isOpen={plan !== null}
      onOpenChange={onOpenChange}
      title={currentPlanName ? `Change to ${plan?.name ?? "this plan"}` : "Confirm plan"}
      subTitle={
        plan?.amount !== null && plan?.amount !== undefined
          ? `${formatDollars(plan.amount)}/${plan.interval ?? "month"}, billed now.`
          : undefined
      }
      footer={
        <div className="flex w-full flex-col items-center gap-2">
          <Button
            type="button"
            variant="primary"
            size="xlg"
            className="w-full rounded-lg"
            loading={isLoading}
            onClick={onConfirm}
          >
            {currentPlanName ? "Confirm change" : "Subscribe"}
          </Button>
          {note ? <p className="text-center text-[12px] text-gray-9 leading-5">{note}</p> : null}
        </div>
      }
    >
      <div className="text-[13px] text-gray-11 leading-6">
        {currentPlanName
          ? `You're moving from ${currentPlanName} to ${plan?.name ?? "the selected plan"}.`
          : `You're subscribing to ${plan?.name ?? "the selected plan"}.`}
      </div>
    </DialogContainer>
  );
}
