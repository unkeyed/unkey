"use client";

import type { Deployment } from "@/lib/collections/deploy/deployments";
import { trpc } from "@/lib/trpc/client";
import {
  CloudUp,
  Earth,
  Hammer2,
  LayerFront,
  Lock,
  Pulse,
  ShieldAlert,
  Sparkle3,
} from "@unkey/icons";
import { Button, SettingCardGroup } from "@unkey/ui";
import { useProjectData } from "../../../data-provider";
import { DeploymentStep } from "./deployment-step";

export function DeploymentApproval({ deployment }: { deployment: Deployment }) {
  const { refetchDeployments, project } = useProjectData();

  const authorize = trpc.deploy.deployment.authorize.useMutation({
    onSuccess: () => {
      refetchDeployments();
    },
  });

  const prUrl =
    project?.repositoryFullName && deployment.prNumber
      ? `https://github.com/${project.repositoryFullName}/pull/${deployment.prNumber}`
      : null;

  return (
    <div className="flex flex-col gap-5">
      <SettingCardGroup>
        <DeploymentStep
          icon={<ShieldAlert iconSize="sm-medium" className="size-[18px]" />}
          title="Authorization Required"
          description="Awaiting team member authorization to proceed"
          status="started"
          statusIcon={<Lock iconSize="sm-medium" className="text-gray-9" />}
        />
        <DeploymentStep
          icon={<LayerFront iconSize="sm-medium" className="size-[18px]" />}
          title="Deployment Queued"
          description="Waiting for authorization"
          status="pending"
        />
        <DeploymentStep
          icon={<Pulse iconSize="sm-medium" className="size-[18px]" />}
          title="Deployment Starting"
          description="Waiting for authorization"
          status="pending"
        />
        <DeploymentStep
          icon={<Hammer2 iconSize="sm-medium" className="size-[18px]" />}
          title="Building Image"
          description="Waiting for authorization"
          status="pending"
        />
        <DeploymentStep
          icon={<CloudUp iconSize="sm-medium" className="size-[18px]" />}
          title="Deploying Containers"
          description="Waiting for authorization"
          status="pending"
        />
        <DeploymentStep
          icon={<Earth iconSize="sm-medium" className="size-[18px]" />}
          title="Assigning Domains"
          description="Waiting for authorization"
          status="pending"
        />
        <DeploymentStep
          icon={<Sparkle3 iconSize="sm-medium" className="size-[18px]" />}
          title="Deployment Finalizing"
          description="Waiting for authorization"
          status="pending"
        />
      </SettingCardGroup>

      {/* Authorization banner */}
      <div className="border border-warningA-4 bg-warningA-2 rounded-[14px] p-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="flex flex-col gap-0.5">
            <span className="text-sm font-medium text-gray-12">Authorization required</span>
            <span className="text-xs text-gray-11">
              An external contributor pushed a commit. A team member must authorize this deployment.
              {prUrl && (
                <>
                  {" "}
                  <a
                    href={prUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="underline hover:text-gray-12 transition-colors"
                  >
                    View Pull Request #{deployment.prNumber}
                  </a>
                </>
              )}
            </span>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="primary"
            size="sm"
            loading={authorize.isLoading}
            onClick={() => authorize.mutate({ deploymentId: deployment.id })}
            className="px-3"
          >
            Authorize
          </Button>
        </div>
      </div>

      {authorize.error && (
        <div className="border border-errorA-4 bg-errorA-2 rounded-[14px] p-4">
          <p className="text-sm text-error-11">{authorize.error.message}</p>
        </div>
      )}
    </div>
  );
}
