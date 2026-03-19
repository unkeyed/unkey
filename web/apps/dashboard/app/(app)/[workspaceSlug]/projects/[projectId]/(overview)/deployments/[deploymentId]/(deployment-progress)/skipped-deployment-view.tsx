"use client";

import { Ban, Github } from "@unkey/icons";
import { Button } from "@unkey/ui";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useProjectData } from "../../../data-provider";

export function SkippedDeploymentView() {
  const { projectId } = useProjectData();
  const params = useParams();
  const workspaceSlug = params.workspaceSlug as string;

  return (
    <div className="flex flex-col items-center gap-8 py-12 animate-fade-slide-in">
      {/* Flow diagram */}
      <div className="flex items-center gap-0">
        {/* GitHub circle */}
        <div className="size-12 rounded-full border border-gray-6 bg-gray-2 flex items-center justify-center">
          <Github iconSize="xl-regular" className="text-gray-12" />
        </div>

        {/* Dashed line */}
        <div className="w-12 border-t border-dashed border-gray-6" />

        {/* Skip circle */}
        <div className="size-12 rounded-full border border-gray-6 bg-gray-2 flex items-center justify-center">
          <Ban iconSize="xl-regular" className="text-gray-9" />
        </div>
      </div>

      {/* Title and description */}
      <div className="flex flex-col items-center gap-2 max-w-md text-center">
        <h2 className="text-lg font-semibold text-gray-12">Deployment Skipped</h2>
        <p className="text-sm text-gray-9">
          No changed files matched the configured watch paths. This deployment was automatically
          skipped.
        </p>
      </div>

      {/* Settings link */}
      <Link href={`/${workspaceSlug}/projects/${projectId}/settings`}>
        <Button variant="ghost" size="sm">
          Configure Watch Paths
        </Button>
      </Link>
    </div>
  );
}
