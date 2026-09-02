"use client";

import { Button, SettingCardGroup } from "@unkey/ui";
import Link from "next/link";
import { useParams } from "next/navigation";
import {
  IconBanOutline18,
  IconChartActivityOutline18,
  IconCloudUploadOutline18,
  IconEarthOutline18,
  IconHammer2Outline18,
  IconLayerFrontOutline18,
  IconSparkle3Outline18,
} from "nucleo-ui-outline-18";
import { routes } from "@/lib/navigation/routes";
import { useProjectData } from "../../../data-provider";
import { useDeployment } from "../layout-provider";
import { DeploymentStep } from "./deployment-step";

export function DeploymentSkipped() {
  const { projectId } = useProjectData();
  const { deployment } = useDeployment();
  const params = useParams();
  const workspaceSlug = params.workspaceSlug as string;

  return (
    <div className="flex flex-col gap-5">
      <SettingCardGroup>
        <DeploymentStep
          icon={<IconBanOutline18 />}
          title="Deployment Skipped"
          description="This deployment was skipped based on your build settings"
          status="started"
          statusIcon={<IconBanOutline18 className="size-4 text-gray-9" />}
        />
        <DeploymentStep
          icon={<IconLayerFrontOutline18 />}
          title="Deployment Queued"
          description="Skipped"
          status="skipped"
        />
        <DeploymentStep
          icon={<IconChartActivityOutline18 />}
          title="Deployment Starting"
          description="Skipped"
          status="skipped"
        />
        <DeploymentStep
          icon={<IconHammer2Outline18 />}
          title="Building Image"
          description="Skipped"
          status="skipped"
        />
        <DeploymentStep
          icon={<IconCloudUploadOutline18 />}
          title="Deploying Containers"
          description="Skipped"
          status="skipped"
        />
        <DeploymentStep
          icon={<IconEarthOutline18 />}
          title="Assigning Domains"
          description="Skipped"
          status="skipped"
        />
        <DeploymentStep
          icon={<IconSparkle3Outline18 />}
          title="Deployment Finalizing"
          description="Skipped"
          status="skipped"
        />
      </SettingCardGroup>

      <div className="border border-grayA-4 bg-grayA-2 rounded-lg p-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="flex flex-col gap-0.5">
            <span className="text-sm font-medium text-gray-12">Deployment skipped</span>
            <span className="text-xs text-gray-11">
              {deployment.triggerReason ??
                "This deployment was skipped. Check your watch paths and auto deploy settings."}
            </span>
          </div>
        </div>
        <Link href={routes.projects.settings({ workspaceSlug, projectId })}>
          <Button variant="primary" size="sm" className="px-3 shrink-0">
            Go to Settings
          </Button>
        </Link>
      </div>
    </div>
  );
}
