"use client";
import { getCurrentVariant } from "@/hooks/use-navbar-variant";
import { PRODUCT_HOME_ROUTES, useProductSelection } from "@/hooks/use-product-selection";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

export default function WorkspacePage() {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();

  const { product } = useProductSelection();

  useEffect(() => {
    // v2 / v3 are flagship-Deploy variants — they skip the product selector
    // and route the workspace root straight to /projects.
    const variant = getCurrentVariant();
    if (variant === "v2" || variant === "v3") {
      router.replace(`/${workspace.slug}/projects`);
      return;
    }
    router.replace(`/${workspace.slug}/${PRODUCT_HOME_ROUTES[product]}`);
  }, [router, workspace.slug, product]);

  return null;
}
