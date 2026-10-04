"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { Button } from "@unkey/ui";
import Link from "next/link";
import { useDeployment } from "./layout-provider";

export function ConnectionPinsNotice() {
  const { deployment } = useDeployment();
  const workspace = useWorkspaceNavigation();
  const { data, isError, refetch } = trpc.deploy.deployment.listConnectionPins.useQuery({
    deploymentId: deployment.id,
    projectId: deployment.projectId,
  });

  if (isError) {
    return (
      <div role="alert" className="flex items-center gap-3 text-sm text-gray-11">
        Could not load the connections that pin this deployment.
        <Button variant="outline" size="sm" onClick={() => refetch()}>
          Retry
        </Button>
      </div>
    );
  }

  if (!data?.pins.length) {
    return null;
  }

  const running = deployment.desiredState === "running";
  return (
    <section className="rounded-lg border border-grayA-4 bg-grayA-2 overflow-hidden">
      <div className="flex items-start justify-between gap-4 px-5 py-4 border-b border-grayA-4">
        <div className="space-y-1">
          <h2 className="text-sm font-medium text-gray-12">
            {running ? "Kept running by connections" : "Connections pin this deployment"}
          </h2>
          <p className="text-xs leading-5 text-gray-10 max-w-2xl">
            Explicit deployment connections have no automatic expiry. Removing a current rule does
            not revoke copies already saved in caller deployments.
          </p>
        </div>
      </div>
      <div className="divide-y divide-grayA-4">
        {data.pins.map((pin) => {
          const connectionsUrl = routes.projects.apps.connections({
            workspaceSlug: workspace.slug,
            projectId: deployment.projectId,
            appId: pin.callerAppId,
          });
          const deploymentUrl =
            pin.provenance === "deployment_snapshot"
              ? routes.projects.apps.deployment({
                  workspaceSlug: workspace.slug,
                  projectId: deployment.projectId,
                  appId: pin.callerAppId,
                  deploymentId: pin.callerDeploymentId,
                })
              : null;

          return (
            <div
              key={`${pin.provenance}-${pin.connectionId}-${
                pin.provenance === "deployment_snapshot" ? pin.callerDeploymentId : "rule"
              }`}
              className="flex items-center justify-between gap-4 px-5 py-3"
            >
              <div className="min-w-0">
                <p className="text-sm text-gray-12 truncate">
                  {running ? "Kept running by connection " : "Pinned by connection "}
                  <span className="font-medium">{pin.connectionName}</span>
                </p>
                <p className="text-xs text-gray-10 mt-0.5">
                  {pin.callerAppName}
                  {pin.provenance === "deployment_snapshot"
                    ? ` · Saved in deployment ${pin.callerDeploymentId} · ${pin.callerStatus}`
                    : " · Current connection rule"}
                </p>
              </div>
              <div className="flex items-center gap-2 shrink-0">
                {deploymentUrl && (
                  <Button variant="outline" size="sm" render={<Link href={deploymentUrl} />}>
                    View deployment
                  </Button>
                )}
                <Button variant="outline" size="sm" render={<Link href={connectionsUrl} />}>
                  View connections
                </Button>
              </div>
            </div>
          );
        })}
      </div>
      {data.truncated && (
        <p className="px-5 py-3 border-t border-grayA-4 text-xs text-gray-10">
          Showing the first 10 pins.
        </p>
      )}
    </section>
  );
}
