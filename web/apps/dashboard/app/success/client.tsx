"use client";
import { routes } from "@/lib/navigation/routes";
import dynamic from "next/dynamic";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

const PlanSelectionModal = dynamic(
  () =>
    import("../(app)/[workspaceSlug]/settings/billing/components/plan-selection-modal").then(
      (mod) => ({
        default: mod.PlanSelectionModal,
      }),
    ),
  {
    ssr: false,
    loading: () => null,
  },
);

type Props = {
  workSpaceSlug?: string;
  showPlanSelection?: boolean;
  products?: Array<{
    id: string;
    name: string;
    priceId: string;
    dollar: number;
    quotas: {
      requestsPerMonth: number;
    };
  }>;
  /**
   * What the user was doing when they got sent through checkout (set by the
   * two-product billing page). Carried back to the billing page, which opens
   * the matching plan picker; its presence disables the legacy forced modal.
   */
  intent?: string;
  /**
   * For the "deploy" intent: the plan the user picked and where they started.
   * Carried back to the projects page, which subscribes and (when from is
   * "create") reopens the create dialog.
   */
  plan?: string;
  from?: string;
};

export function SuccessClient({
  workSpaceSlug,
  showPlanSelection,
  products,
  intent,
  plan,
  from,
}: Props) {
  const router = useRouter();
  const [showModal, setShowModal] = useState(!!(showPlanSelection && products && workSpaceSlug));

  useEffect(() => {
    // If showing modal, don't redirect
    if (showPlanSelection && products && workSpaceSlug) {
      return;
    }

    // Redirect based on workspace availability
    if (workSpaceSlug) {
      if (intent === "deploy") {
        const params = new URLSearchParams();
        if (plan) {
          params.set("pendingPlan", plan);
        }
        if (from) {
          params.set("from", from);
        }
        const query = params.toString();
        router.push(
          `${routes.projects.list({ workspaceSlug: workSpaceSlug })}${query ? `?${query}` : ""}`,
        );
        return;
      }

      router.push(
        `${routes.settings.billing({ workspaceSlug: workSpaceSlug })}${
          intent ? `?intent=${encodeURIComponent(intent)}` : ""
        }`,
      );
    } else {
      router.push("/");
    }
  }, [router, workSpaceSlug, showPlanSelection, products, intent, plan, from]);

  if (showPlanSelection && products && workSpaceSlug) {
    return (
      <PlanSelectionModal
        isOpen={showModal}
        onOpenChange={setShowModal}
        products={products}
        workspaceSlug={workSpaceSlug}
      />
    );
  }

  return <></>;
}
