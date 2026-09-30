"use client";

import {
  CanvasCard,
  CanvasCardHeader,
  DetailList,
  DetailRow,
} from "@/app/(app)/[workspaceSlug]/projects/_components/canvas/primitives";
import type { Deployment } from "@/lib/collections";
import { routes } from "@/lib/navigation/routes";
import {
  Github,
  IconCubeOutline18,
  IconHardDriveOutline18,
  IconLayers2Outline18,
  IconLocation2Outline18,
  IconMicrochipOutline18,
  IconTerminalOutline18,
} from "@unkey/icons";
import { Card } from "@unkey/ui";
import { useAppCurrentDeployment } from "../../../hooks/use-app-current-deployment";
import { useAppScope } from "../../environment-context";
import { AppCanvas } from "./app-canvas";
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
  const icon =
    app?.sourceType === "oci" ? (
      <IconLayers2Outline18 />
    ) : app?.repositoryFullName ? (
      <Github />
    ) : (
      <IconTerminalOutline18 />
    );
  return (
    <CanvasCard>
      <CanvasCardHeader
        icon={icon ?? <IconCubeOutline18 />}
        title={app?.name ?? "App"}
        right={
          <span className="inline-flex items-center gap-1.5 text-xs text-gray-9">
            <span className="size-1.5 rounded-full bg-gray-7" />
            Never deployed
          </span>
        }
      />
      <DetailList>
        <DetailRow icon={<IconLocation2Outline18 />} label="Regions" value="—" />
        <DetailRow icon={<IconHardDriveOutline18 />} label="Instances" value="—" />
        <DetailRow icon={<IconMicrochipOutline18 />} label="Resources" value="—" />
      </DetailList>
    </CanvasCard>
  );
}
