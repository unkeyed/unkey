"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { ChevronLeft } from "@unkey/icons";
import Link from "next/link";
import type React from "react";

interface ResourceHeadingProps {
  resourceType: "api" | "project" | "namespace";
  resourceId: string;
  resourceName?: string; // Optional - not used anymore since name is in nav
}

const RESOURCE_TYPE_PLURAL: Record<ResourceHeadingProps["resourceType"], string> = {
  api: "APIs",
  project: "Projects",
  namespace: "Namespaces",
};

const RESOURCE_TYPE_ROUTES: Record<ResourceHeadingProps["resourceType"], string> = {
  api: "apis",
  project: "projects",
  namespace: "ratelimits", // Namespaces are accessed via /ratelimits route
};

export const ResourceHeading: React.FC<ResourceHeadingProps> = ({ resourceType }) => {
  const workspace = useWorkspaceNavigation();

  // Build back link based on resource type
  const backLink = {
    href: `/${workspace.slug}/${RESOURCE_TYPE_ROUTES[resourceType]}`,
    label: `Back to All ${RESOURCE_TYPE_PLURAL[resourceType]}`,
  };

  return (
    <div className="flex flex-col gap-2 py-2 px-2">
      {/* Back link */}
      <Link
        href={backLink.href}
        className="flex items-center gap-1.5 text-xs text-gray-11 hover:text-gray-12 transition-colors group"
      >
        <ChevronLeft className="w-3 h-3 group-hover:-translate-x-0.5 transition-transform" />
        <span>{backLink.label}</span>
      </Link>
    </div>
  );
};
