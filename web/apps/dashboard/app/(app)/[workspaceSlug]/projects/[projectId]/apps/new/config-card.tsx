"use client";

import { CardFrame } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/_components/flow/card";
import { EnvironmentSettingsProvider } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/environment-provider";
import { DeployDataProvider } from "@/lib/deploy-data";
import { match } from "@unkey/match";
import type { ReactNode } from "react";
import { useNewAppFlow } from "./flow";
import { ConfigurePane } from "./panes/configure-pane";
import { ConfigureSkeleton } from "./panes/configure-skeleton";
import { ImagePane } from "./panes/image-pane";
import { PickRepoPane } from "./panes/repository-pane";
import { SourcePane } from "./panes/source-pane";
import { type Card, type CardView, cardCopy, footerlessPanes } from "./wizard-model";

function WithApp({ appId, children }: { appId: string | null; children: ReactNode }) {
  const { projectId } = useNewAppFlow();
  return appId ? (
    <DeployDataProvider projectId={projectId} appId={appId}>
      {children}
    </DeployDataProvider>
  ) : (
    children
  );
}

function CardBody({ card }: { card: Card }) {
  return match(card)
    .with({ pane: "method" }, ({ selected }) => <SourcePane selected={selected} />)
    .with({ pane: "pick-repo" }, ({ appId }) => (
      <WithApp appId={appId}>
        <PickRepoPane appId={appId} />
      </WithApp>
    ))
    .with({ pane: "image" }, ({ appId }) => (
      <WithApp appId={appId}>
        <ImagePane appId={appId} />
      </WithApp>
    ))
    .with({ pane: "configure" }, ({ appId, source }) => (
      <WithApp appId={appId}>
        <EnvironmentSettingsProvider fallback={<ConfigureSkeleton source={source} />}>
          <ConfigurePane source={source} />
        </EnvironmentSettingsProvider>
      </WithApp>
    ))
    .exhaustive();
}

export function ConfigCard({ view: { card, label }, back }: { view: CardView; back: ReactNode }) {
  return (
    <CardFrame
      copy={cardCopy[card.pane]}
      eyebrow={label}
      back={back}
      footerHidden={footerlessPanes.has(card.pane)}
    >
      <CardBody card={card} />
    </CardFrame>
  );
}
