"use client";

import { cn } from "@/lib/utils";
import { ChevronLeft } from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";
import Link from "next/link";
import type React from "react";

interface ResourceHeadingProps {
  resourceType: "api" | "project" | "namespace";
  resourceName: string;
  resourceId: string;
  backLink: {
    label: string; // e.g., "Back to All APIs"
    href: string;
  };
}

const RESOURCE_TYPE_LABELS: Record<ResourceHeadingProps["resourceType"], string> = {
  api: "API",
  project: "Project",
  namespace: "Namespace",
};

export const ResourceHeading: React.FC<ResourceHeadingProps> = ({
  resourceType,
  resourceName,
  resourceId,
  backLink,
}) => {
  return (
    <div className="flex flex-col gap-2 px-2 py-2">
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
            <div className="flex flex-col">
              <span className="text-xs text-gray-10 uppercase tracking-wide font-medium">
                {RESOURCE_TYPE_LABELS[resourceType]}
              </span>
              <span
                className={cn(
                  "text-sm font-semibold text-gray-12 truncate max-w-[200px]",
                  "cursor-default",
                )}
              >
                {resourceName}
              </span>
            </div>
          </TooltipTrigger>
          <TooltipContent side="right" sideOffset={8}>
            <div className="flex flex-col gap-1">
              <p className="text-xs font-medium">{resourceName}</p>
              <p className="text-xs text-gray-11 font-mono">{resourceId}</p>
            </div>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    </div>
  );
};
