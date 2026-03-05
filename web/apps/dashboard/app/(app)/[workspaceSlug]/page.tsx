"use client";
import { PRODUCT_HOME_ROUTES, useProductSelection } from "@/hooks/use-product-selection";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

export default function WorkspacePage() {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();

  const { product } = useProductSelection();

  useEffect(() => {
    router.replace(`/${workspace.slug}/${PRODUCT_HOME_ROUTES[product]}`);
  }, [router, workspace.slug, product]);

  return null;
}
