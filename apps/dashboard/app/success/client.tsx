"use client";
import dynamic from "next/dynamic";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

const PlanSelectionModal = dynamic(
  () =>
    import(
      "../(app)/[workspaceSlug]/settings/billing/components/plan-selection-modal"
    ).then((mod) => ({
      default: mod.PlanSelectionModal,
    })),
  {
    ssr: false,
    loading: () => null,
  }
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

export function SuccessClient({
  workSpaceSlug,
  showPlanSelection,
  products,
}: Props) {
  const router = useRouter();
  const [showModal, setShowModal] = useState(true);

  useEffect(() => {
    if (showPlanSelection && products && workSpaceSlug) {
      // Show plan selection modal for first-time users
      setShowModal(true);
    } else if (workSpaceSlug) {
      // Regular redirect for existing users
      router.push(`/${workSpaceSlug}/settings/billing`);
    } else {
      // Redirect to root when no workspace slug is available
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
