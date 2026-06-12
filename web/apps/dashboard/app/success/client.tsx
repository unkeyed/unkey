"use client";
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
};

export function SuccessClient({ workSpaceSlug, showPlanSelection, products, intent }: Props) {
  const router = useRouter();
  const [showModal, setShowModal] = useState(!!(showPlanSelection && products && workSpaceSlug));

  useEffect(() => {
    // If showing modal, don't redirect
    if (showPlanSelection && products && workSpaceSlug) {
      return;
    }

    // Redirect based on workspace availability
    if (workSpaceSlug) {
      router.push(
        `/${workSpaceSlug}/settings/billing${intent ? `?intent=${encodeURIComponent(intent)}` : ""}`,
      );
    } else {
      router.push("/");
    }
  }, [router, workSpaceSlug, showPlanSelection, products, intent]);

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
