"use client";

import {
  CanvasCard,
  CanvasCardHeader,
} from "@/app/(app)/[workspaceSlug]/projects/_components/canvas/primitives";
import type { Deployment } from "@/lib/collections";
import { routes } from "@/lib/navigation/routes";
import {
  Github,
  IconCubeOutline18,
  IconEarthOutline18,
  IconLayers2Outline18,
  IconPlusOutline18,
  IconTerminalOutline18,
} from "@unkey/icons";
import { Button, Card } from "@unkey/ui";
import { useAppCurrentDeployment } from "../../../hooks/use-app-current-deployment";
import { useAppScope } from "../../environment-context";
import { CreateDeploymentButton } from "../../navigations/create-deployment-button";
import { AppCanvas } from "./app-canvas";
import { CARD_HEADER_CLASS, CardEyebrow } from "./card-header";
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
      <div className={CARD_HEADER_CLASS}>
        <div className="flex items-center gap-2 min-w-0">
          <CardEyebrow>Never deployed</CardEyebrow>
          <span className="font-mono text-base font-semibold text-gray-9 truncate">
            Domain pending
          </span>
        </div>
        <CreateDeploymentButton
          renderTrigger={({ onClick }) => (
            <Button variant="outline" size="sm" onClick={onClick}>
              <IconPlusOutline18 />
              Create deployment
            </Button>
          )}
        />
      </div>
      <AppCanvas domains={[]} emptyDomain={<PendingDomainCard />} app={<PendingAppNode />} />
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

function PendingDomainCard() {
  return (
    <CanvasCard>
      <CanvasCardHeader
        icon={<IconEarthOutline18 />}
        title="Domain pending"
        mono
        right={<span className="text-xs text-gray-9">Assigned on first deploy</span>}
      />
    </CanvasCard>
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
      <div className="flex flex-col py-1">
        {["Regions", "Instances", "Resources"].map((label) => (
          <div
            key={label}
            className="grid grid-cols-[72px_minmax(0,1fr)] items-start gap-x-3 px-3 py-1.5 text-xs"
          >
            <span className="text-gray-9">{label}</span>
            <span className="text-gray-11">—</span>
          </div>
        ))}
      </div>
    </CanvasCard>
  );
}
