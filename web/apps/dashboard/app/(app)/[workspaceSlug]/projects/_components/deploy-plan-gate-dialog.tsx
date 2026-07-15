"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { type DeployCheckoutOrigin, routes } from "@/lib/navigation/routes";
import type { DeployPlan } from "@/lib/stripe/deployPlan";
import { trpc } from "@/lib/trpc/client";
import type { DeployPlanOption } from "@/lib/trpc/routers/stripe/getDeployPlans";
import type { Route } from "next";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  ComputePlanDialog,
  ComputePlanFeatures,
  ComputePlanRows,
  ComputePlansMoreInfo,
} from "../../settings/billing/components/compute-plan-picker";

type ViewProps = {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  plans: DeployPlanOption[];
  plansLoading: boolean;
  isAdmin: boolean;
  billingHref: Route;
  onSelect: (plan: DeployPlan) => void;
};

/** Presentational layer of the paywall. */
function DeployPlanGateDialogView({
  isOpen,
  onOpenChange,
  plans,
  plansLoading,
  isAdmin,
  billingHref,
  onSelect,
}: ViewProps) {
  function renderPlanSection() {
    if (plansLoading) {
      return (
        <div className="flex flex-col gap-2.5" aria-hidden="true">
          <div className="h-[62px] animate-pulse rounded-[11px] border border-gray-4 bg-grayA-2" />
          <div className="h-[62px] animate-pulse rounded-[11px] border border-gray-4 bg-grayA-2" />
          <div className="h-[62px] animate-pulse rounded-[11px] border border-gray-4 bg-grayA-2" />
        </div>
      );
    }
    if (plans.length === 0) {
      return (
        <div className="rounded-[11px] border border-gray-4 bg-gray-1 px-4 py-6 text-center">
          <p className="text-[13px] text-gray-11">Compute plans aren't available right now.</p>
          <Link
            href={billingHref}
            onClick={() => onOpenChange(false)}
            className="mt-2 inline-block font-medium text-[13px] text-info-11 hover:underline"
          >
            Go to billing
          </Link>
        </div>
      );
    }
    return (
      <ComputePlanRows
        plans={plans}
        onSelect={onSelect}
        disabledReason={isAdmin ? undefined : "Only workspace admins can manage billing."}
      />
    );
  }

  return (
    <ComputePlanDialog
      isOpen={isOpen}
      onOpenChange={onOpenChange}
      title="Choose a Compute plan"
      subTitle="Deploying on Unkey requires a Compute plan. Select one to continue."
    >
      <div className="flex flex-col gap-2.5">
        <ComputePlanFeatures />
        <ComputePlansMoreInfo />
      </div>
      <div className="mt-0">{renderPlanSection()}</div>
      {isAdmin ? null : (
        <p className="text-center text-[12px] text-gray-11">
          Only workspace admins can manage billing.
        </p>
      )}
    </ComputePlanDialog>
  );
}

type Props = {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  /** Where the dialog was opened from, carried through the round-trip. */
  from: DeployCheckoutOrigin;
};

/**
 * The Compute paywall on the projects page. Selecting a plan routes it to
 * payment rather than subscribing inline: with a card on file it hands off to
 * the projects landing (?pendingPlan&from), where usePendingSubscribe runs
 * subscribeDeploy; without one it sends the user to Stripe checkout first,
 * and /success returns them to the same landing. ctrl-api remains the real
 * gate, so the non-admin lockout here is UX only.
 */
export function DeployPlanGateDialog({ isOpen, onOpenChange, from }: Props) {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();

  const { data: plansData, isLoading: plansLoading } = trpc.stripe.getDeployPlans.useQuery(
    undefined,
    { staleTime: 60_000 },
  );
  const { data: currentUser } = trpc.user.getCurrentUser.useQuery();
  const isAdmin = currentUser?.role === "admin";
  const plans = plansData?.plans ?? [];
  const hasPaymentMethod = Boolean(workspace.stripeCustomerId);

  const handleSelect = (plan: DeployPlan) => {
    onOpenChange(false);

    if (hasPaymentMethod) {
      // Card on file: skip Stripe and subscribe on the projects landing.
      router.push(routes.projects.pendingSubscribe({ workspaceSlug: workspace.slug, plan, from }));
    } else {
      router.push(
        routes.settings.stripe.checkout({
          workspaceSlug: workspace.slug,
          intent: "deploy",
          plan,
          from,
        }),
      );
    }
  };

  return (
    <DeployPlanGateDialogView
      isOpen={isOpen}
      onOpenChange={onOpenChange}
      plans={plans}
      plansLoading={plansLoading}
      isAdmin={isAdmin}
      billingHref={routes.settings.billing({ workspaceSlug: workspace.slug })}
      onSelect={handleSelect}
    />
  );
}
