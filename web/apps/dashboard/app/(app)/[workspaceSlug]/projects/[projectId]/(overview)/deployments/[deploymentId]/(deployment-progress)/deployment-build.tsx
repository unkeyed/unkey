"use client";

import { trpc } from "@/lib/trpc/client";
import { Hammer2 } from "@unkey/icons";
import { Button, SettingCardGroup } from "@unkey/ui";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useProjectData } from "../../../data-provider";
import { useDeployment } from "../layout-provider";
import { DeploymentBuildStepsTable } from "./build-steps-table/deployment-build-steps-table";
import { DeploymentStep } from "./deployment-step";

export function DeploymentBuild() {
  const { deployment } = useDeployment();
  const { projectId } = useProjectData();
  const router = useRouter();
  const params = useParams();
  const workspaceSlug = params.workspaceSlug as string;
  const deploymentUrl = `/${workspaceSlug}/projects/${projectId}/deployments/${deployment.id}`;

  const buildSteps = trpc.deploy.deployment.buildSteps.useQuery(
    {
      deploymentId: deployment.id,
      includeStepLogs: true,
    },
    {
      refetchInterval: 1_000,
    },
  );

  router.prefetch(deploymentUrl);

  return (
    <div className="flex flex-col gap-5">
      <SettingCardGroup>
        <DeploymentStep
          icon={<Hammer2 iconSize="sm-medium" className="size-4.5" />}
          title="Build Logs"
          description="Explore the build output for this deployment"
          status="completed"
          expandable={
            <div className="bg-grayA-2">
              <DeploymentBuildStepsTable
                steps={buildSteps.data?.steps ?? []}
                isLoading={buildSteps.isLoading}
                fixedHeight={750}
              />
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
        <span className="text-gray-10 text-[13px] text-center">
          Continue to view live status, domains, and metrics.
        </span>
      </div>
    </div>
  );
}
