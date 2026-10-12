"use client";

import { routes } from "@/lib/navigation/routes";
import { IconHammer2Outline18 } from "@unkey/icons";
import { Button, SettingCardGroup } from "@unkey/ui";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useProjectData } from "../../../data-provider";
import { useDeployment } from "../layout-provider";
import { DeploymentBuildLogs } from "./build-logs/deployment-build-logs";
import { DeploymentStep } from "./deployment-step";

export function DeploymentBuild() {
  const { deployment } = useDeployment();
  const { projectId } = useProjectData();
  const router = useRouter();
  const params = useParams();
  const workspaceSlug = params.workspaceSlug as string;
  const deploymentUrl = routes.projects.apps.deployment({
    workspaceSlug,
    projectId,
    appId: deployment.appId,
    deploymentId: deployment.id,
  });

  router.prefetch(deploymentUrl);

  return (
    <div className="flex flex-col gap-5">
      <SettingCardGroup>
        <DeploymentStep
          icon={<IconHammer2Outline18 className="size-4.5" />}
          title="Build Logs"
          description="Explore the build output for this deployment"
          status="completed"
          expandable={
            <div className="bg-grayA-2">
              <DeploymentBuildLogs fixedHeight={750} />
            </div>
          }
          defaultExpanded
        />
      </SettingCardGroup>

      <div className="flex w-full gap-4 flex-col">
        <Link href={deploymentUrl}>
          <Button className="w-full" size="xlg">
            Continue to deployment
          </Button>
        </Link>
        <span className="text-gray-10 text-sm text-center">
          Continue to view live status, domains, and metrics.
        </span>
      </div>
    </div>
  );
}
