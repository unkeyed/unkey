"use client";

import { ContextNavigation } from "@/components/navigation/sidebar/context-navigation";
import {
  RESOURCE_TYPE_PLURAL,
  RESOURCE_TYPE_ROUTES,
} from "@/components/navigation/sidebar/navigation-configs";
import type { NavigationContext } from "@/hooks/use-navigation-context";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { ChevronLeft } from "@unkey/icons";
import Link from "next/link";

type Props = {
  context: Extract<NavigationContext, { type: "resource" }>;
};

/**
 * Renders when the user is on a resource route (API / Project / Namespace).
 * Shows a back button to the product list, then delegates sub-nav rendering
 * to the existing `ContextNavigation` component — which already knows how
 * to produce the right items per resource type.
 */
export function LeafSidebarView({ context }: Props) {
  const workspace = useWorkspaceNavigation();
  const listHref = `/${workspace.slug}/${RESOURCE_TYPE_ROUTES[context.resourceType]}`;
  const plural = RESOURCE_TYPE_PLURAL[context.resourceType];

  return (
    <div className="flex flex-col gap-2 px-0 pt-1">
      <Link
        href={listHref}
        className="flex items-center gap-1.5 rounded-md px-2 py-1.5 text-[12px] font-medium text-gray-11 hover:bg-gray-3 hover:text-gray-12"
      >
        <ChevronLeft className="h-3.5 w-3.5" />
        <span>All {plural}</span>
      </Link>
      <ContextNavigation context={context} />
    </div>
  );
}
