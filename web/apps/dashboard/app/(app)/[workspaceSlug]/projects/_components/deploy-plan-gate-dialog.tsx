"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { type DeployCheckoutOrigin, routes } from "@/lib/navigation/routes";
import type { DeployPlan } from "@/lib/stripe/deployPlan";
import { trpc } from "@/lib/trpc/client";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ComputePlanPicker } from "../../settings/billing/components/compute-plan-picker";

type Props = {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  /** Where the dialog was opened from, carried through the round-trip. */
  from: DeployCheckoutOrigin;
};

/**
 * The Compute-plan picker as a standalone gate dialog. It composes
 * ComputePlanPicker but, instead of subscribing inline, routes the chosen plan
 * to payment: when a card is on file it hands off to the projects landing where
 * the subscribe runs; without a card it sends the user to Stripe checkout
 * first. ctrl-api remains the real gate, so the non-admin lockout here is UX
 * only.
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
            href={routes.settings.billing({ workspaceSlug: workspace.slug })}
            onClick={() => onOpenChange(false)}
            className="mt-2 inline-block font-medium text-[13px] text-info-11 hover:underline"
          >
            Go to billing
          </Link>
        </div>
      );
    }
    return <ComputePlanPicker.Rows />;
  }

  return (
    <ComputePlanPicker.Provider
      state={{ plans, currentPlan: null, submittingPlan: null }}
      actions={{ onSelect: handleSelect }}
      meta={{
        disabledReason: isAdmin ? undefined : "Only workspace admins can manage billing.",
      }}
    >
      <ComputePlanPicker.Surface isOpen={isOpen} onOpenChange={onOpenChange}>
        {/* Surface is `relative` — absolutely-position decoration (an SVG,
            a gradient) as a sibling here and it anchors to the dialog. */}
        <ComputePlanPicker.Header
          title="Choose a Compute plan"
          subTitle="Deploying on Unkey requires a Compute plan. Select one to continue."
        />
        <ComputePlanPicker.Body>
          <div className="flex flex-col gap-2.5">
            <ComputePlanPicker.Features />
            <ComputePlanPicker.MoreInfoLink />
          </div>
          <div className="mt-0">{renderPlanSection()}</div>
          {isAdmin ? null : (
            <p className="text-center text-[12px] text-gray-11">
              Only workspace admins can manage billing.
            </p>
          )}
        </ComputePlanPicker.Body>
      </ComputePlanPicker.Surface>
    </ComputePlanPicker.Provider>
  );
}
