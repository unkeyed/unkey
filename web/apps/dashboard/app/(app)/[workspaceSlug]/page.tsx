"use client";
import { PRODUCT_HOME_ROUTES, useProductSelection } from "@/hooks/use-product-selection";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useFlag } from "@/lib/flags/provider";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

export default function WorkspacePage() {
  const newNavigation = useFlag("newNavigation");
  return newNavigation ? <NewWorkspaceHome /> : <LegacyWorkspaceHome />;
}

function NewWorkspaceHome() {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();

  useEffect(() => {
    router.replace(`/${workspace.slug}/projects`);
  }, [router, workspace.slug]);

  return null;
}

function LegacyWorkspaceHome() {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const { product } = useProductSelection();

  useEffect(() => {
    router.replace(`/${workspace.slug}/${PRODUCT_HOME_ROUTES[product]}`);
  }, [router, workspace.slug, product]);

  return null;
}
