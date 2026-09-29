"use client";

import { FEATURE_ICONS } from "@/components/billing/plan-feature-icons";
import { PlanTierIcon } from "@/components/billing/plan-tier-icons";
import { announceUpgrade } from "@/components/billing/upgrade-success/upgrade-success-dialog";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import {
  type CurrentPlan,
  type PlanCardState,
  type PlanPrice,
  isComputeUpgrade,
  planCardStates,
  planName,
} from "@/lib/billing/plan-card-state";
import {
  EVERY_PLAN_INCLUDES,
  type PlanFeatureRow,
  planFeatures,
} from "@/lib/billing/plan-features";
import { formatDollars } from "@/lib/fmt";
import { type DeployCheckoutOrigin, routes } from "@/lib/navigation/routes";
import type { DeployPlan } from "@/lib/stripe/deployPlan";
import { trpc } from "@/lib/trpc/client";
import type { DeployPlanOption } from "@/lib/trpc/routers/stripe/getDeployPlans";
import { IconCheckOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import { Button, DialogContainer, Skeleton, toast } from "@unkey/ui";
import { cn } from "cn";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { CancelComputeDialog, CancelPlanLink } from "./cancel-actions";
import { PLAN_BLURBS } from "./compute-plan-copy";
import { ADMIN_ONLY_TOOLTIP } from "./constants";

const CHANGE_NOTE =
  "Takes effect immediately. Upgrades are charged now and add the difference as usage credits; downgrades keep this period's credits, with the new fee starting next period.";

type ComputePlansProps = {
  plans: readonly DeployPlan[];
  options: DeployPlanOption[] | undefined;
  current: CurrentPlan;
  usageCents: number | null;
  isAdmin: boolean;
  from: DeployCheckoutOrigin;
  manage: boolean;
  onChanged: () => void;
};

export function ComputePlans({
  plans,
  options,
  current,
  usageCents,
  isAdmin,
  from,
  manage,
  onChanged,
}: ComputePlansProps) {
  const workspace = useWorkspaceNavigation();
  const router = useRouter();
  const pathname = usePathname();
  const trpcUtils = trpc.useUtils();
  const [pendingOption, setPendingOption] = useState<DeployPlanOption | null>(null);
  const [checkoutPlan, setCheckoutPlan] = useState<DeployPlan | null>(null);
  const [isCancelOpen, setCancelOpen] = useState(false);

  useEffect(() => {
    // Returning from Stripe via the back button restores this page from the
    // bfcache with the checkout spinner still set.
    const reset = (event: PageTransitionEvent) => {
      if (event.persisted) {
        setCheckoutPlan(null);
      }
    };
    window.addEventListener("pageshow", reset);
    return () => window.removeEventListener("pageshow", reset);
  }, []);

  const change = trpc.stripe.changeDeployPlan.useMutation({
    onSuccess: async (result, { plan }) => {
      if (result.kind === "payment_required") {
        window.location.assign(result.paymentUrl);
        return;
      }
      const isUpgrade = current.status === "plan" && isComputeUpgrade(current.plan, plan);
      if (!isUpgrade) {
        toast.success("Compute plan changed");
      }
      await Promise.all([
        trpcUtils.stripe.getDeploySubscription.invalidate(),
        trpcUtils.stripe.getDeployEntitlement.invalidate(),
        trpcUtils.stripe.getUpcomingInvoice.invalidate(),
        trpcUtils.stripe.getDeployCredit.invalidate(),
        trpcUtils.billing.queryDeployUsage.invalidate(),
        trpcUtils.workspace.getCurrent.invalidate(),
      ]);
      router.refresh();
      setPendingOption(null);
      onChanged();
      if (isUpgrade) {
        announceUpgrade({ kind: "compute", plan });
      }
    },
    onError: (err) => toast.error(err.message),
  });

  const select = (option: DeployPlanOption) => {
    if (current.status === "plan") {
      setPendingOption(option);
      return;
    }
    setCheckoutPlan(option.plan);
    window.location.assign(
      routes.settings.stripe.checkout({
        workspaceSlug: workspace.slug,
        intent: "deploy",
        plan: option.plan,
        from,
        returnTo: pathname,
      }),
    );
  };

  return (
    <div className="flex w-full flex-col gap-6">
      <div
        className={cn(
          "mx-auto grid w-full gap-4",
          plans.length === 2 ? "max-w-[720px] md:grid-cols-2" : "md:grid-cols-3",
        )}
      >
        {planCardStates({ plans, options, current, usageCents }).map((card) => (
          <PlanCard
            key={card.plan}
            card={card}
            isAdmin={isAdmin}
            checkoutPlan={checkoutPlan}
            onSelect={select}
          />
        ))}
      </div>
      <EveryPlanIncludesStrip />
      {current.status === "error" ? (
        <p className="text-center text-gray-11 text-sm">
          Your current plan could not be loaded. Reload the page or contact support@unkey.com.
        </p>
      ) : null}
      {manage && current.status === "plan" ? (
        <div className="flex justify-center">
          <CancelPlanLink isAdmin={isAdmin} onClick={() => setCancelOpen(true)}>
            Cancel Compute plan
          </CancelPlanLink>
        </div>
      ) : null}
      <CancelComputeDialog open={isCancelOpen} onOpenChange={setCancelOpen} />
      {current.status === "plan" ? (
        <ComputePlanConfirmDialog
          plan={pendingOption}
          onOpenChange={(isOpen) => {
            if (!isOpen) {
              setPendingOption(null);
            }
          }}
          onConfirm={() => {
            if (pendingOption) {
              change.mutate({ plan: pendingOption.plan });
            }
          }}
          isLoading={change.isLoading}
          currentPlanName={planName(current.plan, options)}
          isUpgrade={pendingOption !== null && isComputeUpgrade(current.plan, pendingOption.plan)}
        />
      ) : null}
    </div>
  );
}

function PlanCard({
  card,
  isAdmin,
  checkoutPlan,
  onSelect,
}: {
  card: PlanCardState;
  isAdmin: boolean;
  checkoutPlan: DeployPlan | null;
  onSelect: (option: DeployPlanOption) => void;
}) {
  const { action } = card;
  return (
    <div className="flex flex-col gap-5 rounded-xl border bg-raised p-5 md:p-6">
      <div className="flex flex-col gap-1">
        <div className="flex items-center justify-between gap-2">
          <h3 className="flex items-center gap-2 font-medium text-base text-gray-12">
            <PlanTierIcon plan={card.plan} className="size-4" />
            {card.name}
          </h3>
        </div>
        <span className="text-gray-11 text-sm">{PLAN_BLURBS[card.plan]}</span>
      </div>

      <div className="flex h-9 items-baseline gap-1">
        <PlanPriceLabel price={card.price} />
      </div>

      <Button
        variant="primary"
        size="lg"
        className="w-full"
        disabled={action.type !== "select" || !isAdmin || checkoutPlan !== null}
        loading={checkoutPlan === card.plan}
        title={isAdmin ? undefined : ADMIN_ONLY_TOOLTIP}
        onClick={() => {
          if (action.type === "select") {
            onSelect(action.option);
          }
        }}
      >
        {action.label}
      </Button>
      {card.warning ? <p className="text-sm text-warning-11 leading-5">{card.warning}</p> : null}

      <PlanFeatureList rows={planFeatures(card.plan)} />
      <EveryPlanIncludes />
    </div>
  );
}

function PlanPriceLabel({ price }: { price: PlanPrice }) {
  return match(price)
    .with({ type: "loading" }, () => <Skeleton className="h-8 w-20 self-center rounded-md" />)
    .with({ type: "contact" }, () => (
      <span className="font-semibold text-3xl text-gray-12">Contact us</span>
    ))
    .with({ type: "amount" }, ({ cents, interval }) => (
      <>
        <span className="font-semibold text-3xl text-gray-12 tabular-nums">
          {formatDollars(cents)}
        </span>
        <span className="text-gray-11 text-sm">{interval === "year" ? "/yr" : "/mo"}</span>
      </>
    ))
    .exhaustive();
}

function PlanFeatureList({ rows }: { rows: PlanFeatureRow[] }) {
  return (
    <ul className="flex flex-col gap-2.5">
      {rows.map((row) => {
        const Icon = FEATURE_ICONS[row.kind];
        return (
          <li
            key={row.label}
            className={cn(
              "flex items-center gap-2.5 text-sm",
              row.included ? "text-gray-12" : "text-gray-9 line-through",
            )}
          >
            <Icon
              className={cn("size-4 shrink-0", row.included ? "text-gray-11" : "text-gray-8")}
            />
            {row.label}
          </li>
        );
      })}
    </ul>
  );
}

function EveryPlanIncludes() {
  return (
    <div className="mt-auto flex flex-col gap-2.5 border-t pt-4 md:hidden">
      <span className="text-gray-11 text-xs">Every plan includes</span>
      <ul className="flex flex-col gap-2">
        {EVERY_PLAN_INCLUDES.map((feature) => (
          <li key={feature} className="flex items-center gap-2.5 text-gray-11 text-sm">
            <IconCheckOutline18 className="size-4 shrink-0 text-gray-9" />
            {feature}
          </li>
        ))}
      </ul>
    </div>
  );
}

function EveryPlanIncludesStrip() {
  return (
    <div className="hidden flex-wrap items-center justify-center gap-x-6 gap-y-2 text-gray-11 text-sm md:flex">
      <span className="text-gray-11 text-xs">Every plan includes</span>
      {EVERY_PLAN_INCLUDES.map((feature) => (
        <span key={feature} className="flex items-center gap-2">
          <IconCheckOutline18 className="size-4 shrink-0 text-gray-9" />
          {feature}
        </span>
      ))}
    </div>
  );
}

type ComputePlanConfirmDialogProps = {
  plan: DeployPlanOption | null;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
  isLoading: boolean;
  currentPlanName: string;
  isUpgrade: boolean;
};

function ComputePlanConfirmDialog({
  plan,
  onOpenChange,
  onConfirm,
  isLoading,
  currentPlanName,
  isUpgrade,
}: ComputePlanConfirmDialogProps) {
  return (
    <DialogContainer
      isOpen={plan !== null}
      onOpenChange={onOpenChange}
      title={`Change to ${plan?.name ?? "this plan"}`}
      subTitle={
        plan?.amount !== null && plan?.amount !== undefined
          ? `${formatDollars(plan.amount)}/${plan.interval ?? "month"}, ${isUpgrade ? "billed now" : "from next period"}.`
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
            Confirm change
          </Button>
          <p className="text-center text-xs text-gray-9 leading-5">{CHANGE_NOTE}</p>
        </div>
      }
    >
      <div className="text-sm text-gray-11 leading-6">
        {`You're moving from ${currentPlanName} to ${plan?.name ?? "the selected plan"}.`}
      </div>
    </DialogContainer>
  );
}
