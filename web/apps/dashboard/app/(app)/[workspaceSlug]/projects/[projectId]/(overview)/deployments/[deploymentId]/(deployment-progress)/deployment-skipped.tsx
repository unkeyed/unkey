"use client";

import { Ban, CloudUp, Earth, Hammer2, LayerFront, Pulse, Sparkle3 } from "@unkey/icons";
import { Button, SettingCardGroup } from "@unkey/ui";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useProjectData } from "../../../data-provider";
import { DeploymentStep } from "./deployment-step";

export function DeploymentSkipped() {
  const { projectId } = useProjectData();
  const params = useParams();
  const workspaceSlug = params.workspaceSlug as string;

  return (
    <div className="flex flex-col gap-5">
      <SettingCardGroup>
        <DeploymentStep
          icon={<Ban iconSize="sm-medium" className="size-[18px]" />}
          title="Deployment Skipped"
          description="No changed files matched the configured watch paths"
          status="started"
          statusIcon={<Ban className="text-gray-9" iconSize="md-regular" />}
        />
        <DeploymentStep
          icon={<LayerFront iconSize="sm-medium" className="size-[18px]" />}
          title="Deployment Queued"
          description="Skipped"
          status="skipped"
        />
        <DeploymentStep
          icon={<Pulse iconSize="sm-medium" className="size-[18px]" />}
          title="Deployment Starting"
          description="Skipped"
          status="skipped"
        />
        <DeploymentStep
          icon={<Hammer2 iconSize="sm-medium" className="size-[18px]" />}
          title="Building Image"
          description="Skipped"
          status="skipped"
        />
        <DeploymentStep
          icon={<CloudUp iconSize="sm-medium" className="size-[18px]" />}
          title="Deploying Containers"
          description="Skipped"
          status="skipped"
        />
        <DeploymentStep
          icon={<Earth iconSize="sm-medium" className="size-[18px]" />}
          title="Assigning Domains"
          description="Skipped"
          status="skipped"
        />
        <DeploymentStep
          icon={<Sparkle3 iconSize="sm-medium" className="size-[18px]" />}
          title="Deployment Finalizing"
          description="Skipped"
          status="skipped"
        />
      </SettingCardGroup>

      <div className="border border-grayA-4 bg-grayA-2 rounded-[14px] p-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="flex flex-col gap-0.5">
            <span className="text-sm font-medium text-gray-12">Deployment skipped</span>
            <span className="text-xs text-gray-11">
              No changed files matched the configured watch paths.
            </span>
          </div>
        </div>
        <Link href={`/${workspaceSlug}/projects/${projectId}/settings`}>
          <Button variant="primary" size="sm" className="px-3 shrink-0">
            Go to Settings
          </Button>
        </Link>
      </div>
    </div>
  );
}
