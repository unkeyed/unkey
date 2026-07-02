"use client";

import { formatDollars } from "@/lib/fmt";
import type { DeployPlan } from "@/lib/stripe/deployPlan";
import type { DeployPlanOption } from "@/lib/trpc/routers/stripe/getDeployPlans";
import { cn } from "@/lib/utils";
import { ArrowRight, ArrowUpRight, Check, CircleInfo } from "@unkey/icons";
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@unkey/ui";
import { createContext, use } from "react";
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

/**
 * A composed Compute-plan picker built as compound components over a context
 * provider, so each caller (the deploy gate dialog, the billing Compute card)
 * injects its own selection semantics and renders only the pieces it needs.
 * State is dependency-injected: the picker UI never owns it, callers do.
 *
 * Deliberately separate from PlanChangeModal (the API add-on picker): Compute
 * uses enriched stacked rows with a per-row commit button, not radio rows with
 * a single bottom CTA.
 */

interface ComputePlanState {
  plans: DeployPlanOption[];
  /** The plan the workspace is on, or null when subscribing for the first time. */
  currentPlan: DeployPlan | null;
  /** The plan whose mutation is in flight, for the row's loading state. */
  submittingPlan: DeployPlan | null;
}

interface ComputePlanActions {
  onSelect: (plan: DeployPlan) => void;
  /**
   * The commit label for a row, e.g. "Select", "Upgrade", "Downgrade". The
   * directional billing card overrides it; the gate dialog uses the default.
   */
  selectLabel?: (plan: DeployPlanOption) => string;
  /**
   * Optional warning shown under a row, e.g. usage already exceeds its credits.
   * Return null for no warning.
   */
  warningFor?: (plan: DeployPlanOption) => string | null;
}

interface ComputePlanMeta {
  /** Disable every commit button and show this reason (admin-only gate). */
  disabledReason?: string;
  /** Docs link target for the "how credits work" info strip. */
  creditsHref: string;
}

interface ComputePlanContextValue {
  state: ComputePlanState;
  actions: ComputePlanActions;
  meta: ComputePlanMeta;
}

const ComputePlanContext = createContext<ComputePlanContextValue | null>(null);

function useComputePlan(): ComputePlanContextValue {
  const ctx = use(ComputePlanContext);
  if (!ctx) {
    throw new Error(
      "ComputePlanPicker subcomponents must be used within ComputePlanPicker.Provider",
    );
  }
  return ctx;
}

type ProviderProps = {
  state: ComputePlanState;
  actions: ComputePlanActions;
  meta?: Partial<ComputePlanMeta>;
  children: React.ReactNode;
};

function Provider({ state, actions, meta, children }: ProviderProps) {
  return (
    <ComputePlanContext
      value={{
        state,
        actions,
        meta: { creditsHref: CREDITS_LINK_HREF, ...meta },
      }}
    >
      {children}
    </ComputePlanContext>
  );
}

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

function Row({ plan }: { plan: DeployPlanOption }) {
  const { state, actions, meta } = useComputePlan();
  const isCurrent = plan.plan === state.currentPlan;
  const isSubmitting = plan.plan === state.submittingPlan;
  const disabled = isCurrent || isSubmitting || meta.disabledReason !== undefined;
  const blurb = PLAN_BLURBS[plan.plan] ?? plan.description ?? "";
  const warning = !isCurrent && actions.warningFor ? actions.warningFor(plan) : null;
  const label = actions.selectLabel ? actions.selectLabel(plan) : "Select";

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
          title={meta.disabledReason}
          onClick={() => actions.onSelect(plan.plan)}
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

function Rows() {
  const { state } = useComputePlan();
  return (
    <div className="flex flex-col gap-2.5 pt-2">
      {state.plans.map((plan) => (
        <Row key={plan.plan} plan={plan} />
      ))}
    </div>
  );
}

function Features() {
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

function MoreInfoLink() {
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

function AllPlansInclude() {
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

function InfoStrip() {
  const { meta } = useComputePlan();
  return (
    <div className="flex items-start gap-2.5 rounded-[11px] border border-gray-4 bg-gray-1 px-3.5 py-3">
      <CircleInfo iconSize="lg-regular" className="mt-px shrink-0 text-info-9" />
      <p className="text-[12.5px] text-gray-11 leading-relaxed">
        {CREDITS_INFO} {/* @dh todo - add docs */}
        <a
          href={meta.creditsHref}
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

type SurfaceProps = {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  className?: string;
  children: React.ReactNode;
};

/**
 * A headless dialog frame for the picker: the raw Radix content surface with
 * no default header divider, gray content area, or footer box. It only sizes
 * and clips the modal. DialogContent is already `fixed`, so it establishes a
 * containing block — absolutely-position decoration (an SVG, a gradient) as a
 * child and it anchors here without needing an extra `relative`. (Adding
 * `relative` would drop the base `fixed` centering via tailwind-merge.)
 * Compose Header / Body / Rows / etc. inside it and own the layout yourself.
 */
function Surface({ isOpen, onOpenChange, className, children }: SurfaceProps) {
  return (
    <Dialog open={isOpen} onOpenChange={onOpenChange}>
      <DialogContent
        className={cn(
          "flex max-h-[90vh] w-[90%] max-w-[560px] flex-col gap-0 overflow-hidden rounded-2xl! border-gray-4 bg-gray-1 p-0",
          className,
        )}
      >
        {children}
      </DialogContent>
    </Dialog>
  );
}

type HeaderProps = {
  title: string;
  subTitle?: string;
  className?: string;
};

/**
 * Optional title/subtitle for a Surface. Uses Radix DialogTitle/Description so
 * the dialog stays labelled for a11y. Skip it entirely and render your own
 * header if you want fancier chrome.
 */
function Header({ title, subTitle, className }: HeaderProps) {
  return (
    <div className={cn("flex flex-col gap-1.5 px-[22px] pt-6 pb-3.5", className)}>
      <DialogTitle className="font-semibold text-[22px] text-gray-12 leading-none tracking-[-0.03em]">
        {title}
      </DialogTitle>
      {subTitle ? (
        <DialogDescription className="text-[14px] text-gray-11 leading-normal">
          {subTitle}
        </DialogDescription>
      ) : null}
    </div>
  );
}

/**
 * Scrollable content region for a Surface, with the picker's standard padding
 * and vertical rhythm. Optional — compose your own div if you need something
 * different.
 */
function Body({ className, children }: { className?: string; children: React.ReactNode }) {
  return (
    <div className={cn("flex flex-col gap-2.5 overflow-y-auto scrollbar-hide p-6 pt-0", className)}>
      {children}
    </div>
  );
}

export const ComputePlanPicker = {
  Provider,
  Surface,
  Header,
  Body,
  Features,
  MoreInfoLink,
  Rows,
  AllPlansInclude,
  InfoStrip,
};

export type { ComputePlanState, ComputePlanActions, ComputePlanMeta };
