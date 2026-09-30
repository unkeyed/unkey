"use client";

import {
  Node,
  NodeHeader,
  RowItem,
  Rows,
  StatusBadge,
} from "@/app/(app)/[workspaceSlug]/projects/_components/canvas/nodes";
import type { Deployment } from "@/lib/collections";
import { routes } from "@/lib/navigation/routes";
import { IconEarthOutline18, IconLayers3Outline18, IconMicrochipOutline18 } from "@unkey/icons";
import { Card } from "@unkey/ui";
import { useAppCurrentDeployment } from "../../../hooks/use-app-current-deployment";
import { useAppScope } from "../../environment-context";
import { AppCanvas, appIcon } from "./app-canvas";
import { NewerDeploymentRow } from "./card-newer-deployment";

/**
 * The overview card for an environment with nothing ready yet. Same frame as
 * the deployed card so a first deploy fills it in rather than swapping layouts.
 */
export function EnvironmentPendingCard({
  newerDeployment,
}: {
  newerDeployment: Deployment | undefined;
}) {
  const scope = useAppScope();
  return (
    <Card className="relative z-10 flex flex-col">
      <AppCanvas domains={[]} app={<PendingAppNode />} />
      {newerDeployment && (
        <NewerDeploymentRow
          deployment={newerDeployment}
          href={routes.projects.apps.deployment({
            ...scope,
            deploymentId: newerDeployment.id,
            build: true,
          })}
        />
      )}
    </Card>
  );
}

function PendingAppNode() {
  const { app } = useAppCurrentDeployment();
  return (
    <Node edge="border">
      <NodeHeader
        icon={appIcon(app)}
        title={app?.name ?? "App"}
        right={<StatusBadge dotClass="bg-gray-7">Never deployed</StatusBadge>}
      />
      <Rows>
        <RowItem icon={<IconEarthOutline18 />} label="Regions" value="—" />
        <RowItem icon={<IconLayers3Outline18 />} label="Instances" value="—" />
        <RowItem icon={<IconMicrochipOutline18 />} label="Resources" value="—" />
      </Rows>
    </Node>
  );
}
