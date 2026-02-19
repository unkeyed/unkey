"use client";

import { useResourceName } from "@/hooks/use-resource-name";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { cn } from "@/lib/utils";
import { ChevronLeft } from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";
import Link from "next/link";
import type React from "react";

interface ResourceHeadingProps {
  resourceType: "api" | "project" | "namespace";
  resourceId: string;
  resourceName?: string; // Optional - will use resourceId as fallback
}

const RESOURCE_TYPE_LABELS: Record<ResourceHeadingProps["resourceType"], string> = {
  api: "API",
  project: "Project",
  namespace: "Namespace",
};

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

export const ResourceHeading: React.FC<ResourceHeadingProps> = ({
  resourceType,
  resourceId,
  resourceName,
}) => {
  const workspace = useWorkspaceNavigation();

  // Fetch resource name if not provided
  const fetchedResourceName = useResourceName(resourceType, resourceId);

  // Build back link based on resource type
  const backLink = {
    href: `/${workspace.slug}/${RESOURCE_TYPE_ROUTES[resourceType]}`,
    label: `Back to All ${RESOURCE_TYPE_PLURAL[resourceType]}`,
  };

  // Use provided name, fetched name, or fallback to resourceId
  const displayName = resourceName || fetchedResourceName || resourceId;

  return (
    <div className="flex flex-col gap-3 px-2 py-2">
      {/* Back link */}
      <Link
        href={backLink.href}
        className="flex items-center gap-1.5 text-xs text-gray-11 hover:text-gray-12 transition-colors group"
      >
        <ChevronLeft className="w-3 h-3 group-hover:-translate-x-0.5 transition-transform" />
        <span>{backLink.label}</span>
      </Link>

      {/* Resource name with ID in tooltip */}
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <div className="flex flex-col mt-1 w-fit">
              <span className="text-xs text-gray-10 uppercase tracking-wide font-medium">
                {RESOURCE_TYPE_LABELS[resourceType]}
              </span>
              <span
                className={cn(
                  "text-sm font-semibold text-gray-12 truncate max-w-[200px]",
                  "cursor-default",
                )}
              >
                {displayName}
              </span>
            </div>
          </TooltipTrigger>
          <TooltipContent side="right" sideOffset={8}>
            <div className="flex flex-col gap-1">
              <p className="text-xs font-medium">{displayName}</p>
              <p className="text-xs text-gray-11 font-mono">{resourceId}</p>
            </div>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    </div>
  );
};
