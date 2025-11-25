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
};

export function SuccessClient({ workSpaceSlug, showPlanSelection, products }: Props) {
  const router = useRouter();
  const [showModal, setShowModal] = useState(!!(showPlanSelection && products && workSpaceSlug));

  useEffect(() => {
    // If showing modal, don't redirect
    if (showPlanSelection && products && workSpaceSlug) {
      return;
    }

    // Redirect based on workspace availability
    if (workSpaceSlug) {
      router.push(`/${workSpaceSlug}/settings/billing`);
    } else {
      router.push("/");
    }
  }, [router, workSpaceSlug, showPlanSelection, products]);

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
