"use client";

import { useNavbarVariant } from "@/hooks/use-navbar-variant";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { generateFakeDeployments } from "@/lib/fake-deployments";
import { BookBookmark } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { usePathname } from "next/navigation";
import { useProjectData } from "../../data-provider";
import { useDeployments } from "../hooks/use-deployments";
import { DeploymentRow } from "./deployment-row";
import { DeploymentsSkeleton } from "./deployments-skeleton";

export function DeploymentsCardList() {
  const { deployments } = useDeployments();
  const { project, projectId } = useProjectData();
  const currentDeploymentId = project?.currentDeploymentId;
  const workspace = useWorkspaceNavigation();
  const { variant } = useNavbarVariant();
  const pathname = usePathname() ?? "";

  // When inside an app route (`/apps/[slug]/...`) in v2b, deployment
  // rows should drill into the per-app detail route rather than the
  // project-level legacy one.
  const appMatch = pathname.match(/\/projects\/[^/]+\/apps\/([^/]+)/);
  const appSlug = appMatch?.[1];
  const hrefFor = (deploymentId: string) => {
    const base = `/${workspace.slug}/projects/${project?.id}`;
    return appSlug
      ? `${base}/apps/${appSlug}/deployments/${deploymentId}`
      : `${base}/deployments/${deploymentId}`;
  };

  if (deployments.isLoading && !deployments.data) {
    return <DeploymentsSkeleton />;
  }

  const realItems = deployments.data ?? [];

  // Prototype-only: v2b falls back to deterministic fakes so Dave can
  // see the populated layout for projects without real deploys.
  if (realItems.length === 0 && variant === "v2b") {
    const fakes = generateFakeDeployments(projectId);
    return (
      <div className="border border-grayA-4 rounded-[14px] overflow-hidden divide-y divide-grayA-4">
        {fakes.map((deployment) => (
          <DeploymentRow
            key={deployment.id}
            deployment={deployment}
            environment={undefined}
            isCurrent={false}
            isRolledBack={false}
            href={hrefFor(deployment.id)}
          />
        ))}
      </div>
    );
  }

  if (realItems.length === 0) {
    return (
      <div className="border border-grayA-4 rounded-[14px] overflow-hidden">
        <div className="w-full flex justify-center items-center py-16 px-4">
          <Empty className="w-[400px] flex items-start">
            <Empty.Icon className="w-auto" />
            <Empty.Title>No Deployments Found</Empty.Title>
            <Empty.Description className="text-left">
              There are no deployments yet. Push to your connected repository or trigger a manual
              deployment to get started.
            </Empty.Description>
            <Empty.Actions className="mt-4 justify-start">
              <a
                href="https://www.unkey.com/docs/build-and-deploy/deployments"
                target="_blank"
                rel="noopener noreferrer"
              >
                <Button size="md">
                  <BookBookmark />
                  Learn about Deployments
                </Button>
              </a>
            </Empty.Actions>
          </Empty>
        </div>
      </div>
    );
  }

  return (
    <div className="border border-grayA-4 rounded-[14px] overflow-hidden divide-y divide-grayA-4">
      {realItems.map((item) => {
        const { deployment, environment } = item;
        if (!deployment) {
          return null;
        }
        const isCurrent = currentDeploymentId === deployment.id;
        return (
          <DeploymentRow
            key={deployment.id}
            deployment={deployment}
            environment={environment}
            isCurrent={isCurrent}
            isRolledBack={isCurrent && (project?.isRolledBack ?? false)}
            href={hrefFor(deployment.id)}
          />
        );
      })}
    </div>
  );
}
