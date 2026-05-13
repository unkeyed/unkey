"use client";

import type { Deployment } from "@/lib/collections/deploy/deployments";
import { Ban, CloudUp, Earth, Hammer2, LayerFront, Pulse, Sparkle3 } from "@unkey/icons";
import { Button, SettingCardGroup } from "@unkey/ui";
import { useState } from "react";
import { RedeployDialog } from "../../components/table/components/actions/redeploy-dialog";
import type { StepsData } from "./deployment-progress";
import { DeploymentStep } from "./deployment-step";

const STEP_ORDER: Array<{
  key: keyof NonNullable<StepsData>;
  icon: React.ReactNode;
  title: string;
}> = [
  {
    key: "queued",
    icon: <LayerFront iconSize="sm-medium" className="size-[18px]" />,
    title: "Deployment Queued",
  },
  {
    key: "starting",
    icon: <Pulse iconSize="sm-medium" className="size-[18px]" />,
    title: "Deployment Starting",
  },
  {
    key: "building",
    icon: <Hammer2 iconSize="sm-medium" className="size-[18px]" />,
    title: "Building Image",
  },
  {
    key: "deploying",
    icon: <CloudUp iconSize="sm-medium" className="size-[18px]" />,
    title: "Deploying Containers",
  },
  {
    key: "network",
    icon: <Earth iconSize="sm-medium" className="size-[18px]" />,
    title: "Assigning Domains",
  },
  {
    key: "finalizing",
    icon: <Sparkle3 iconSize="sm-medium" className="size-[18px]" />,
    title: "Deployment Finalizing",
  },
];

type StopReason = "cancelled" | "superseded";

type DeploymentCancelledProps = {
  deployment: Deployment;
  stepsData?: StepsData;
  reason: StopReason;
};

const COPY: Record<StopReason, { step: string; title: string; description: string }> = {
  cancelled: {
    step: "Cancelled",
    title: "Deployment cancelled",
    description: "You aborted this deployment. Redeploy to try again.",
  },
  superseded: {
    step: "Superseded",
    title: "Deployment superseded",
    description: "A newer commit on this branch replaced this deployment.",
  },
};

export function DeploymentCancelled({ deployment, stepsData, reason }: DeploymentCancelledProps) {
  const [redeployOpen, setRedeployOpen] = useState(false);
  const copy = COPY[reason];

  // Find the first step that has an error and isn't completed -- that's
  // the step that was actively running when the deployment was stopped.
  const stoppedKey = STEP_ORDER.find(
    ({ key }) => stepsData?.[key]?.error && !stepsData?.[key]?.completed,
  )?.key;

  return (
    <div className="flex flex-col gap-5">
      <SettingCardGroup>
        {STEP_ORDER.map(({ key, icon, title }) => {
          const step = stepsData?.[key];
          const isStoppedHere = key === stoppedKey;

          // Every step in the cancelled view shows the same description so
          // the page reads as a single intent: this deployment was stopped.
          // Steps that finished before the cancel keep their duration so
          // you can still see how far it got.
          return (
            <DeploymentStep
              key={key}
              icon={icon}
              title={title}
              description={copy.step}
              duration={step?.duration ?? undefined}
              status="skipped"
              statusIcon={
                isStoppedHere ? <Ban className="text-gray-9" iconSize="md-regular" /> : undefined
              }
            />
          );
        })}
      </SettingCardGroup>

      <div className="border border-grayA-4 bg-grayA-2 rounded-[14px] p-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="flex flex-col gap-0.5">
            <span className="text-sm font-medium text-gray-12">{copy.title}</span>
            <span className="text-xs text-gray-11">{copy.description}</span>
          </div>
        </div>
        <Button
          variant="primary"
          size="sm"
          onClick={() => setRedeployOpen(true)}
          className="px-3 shrink-0"
        >
          Redeploy
        </Button>
      </div>

      <RedeployDialog
        isOpen={redeployOpen}
        onClose={() => setRedeployOpen(false)}
        selectedDeployment={deployment}
      />
    </div>
  );
}
