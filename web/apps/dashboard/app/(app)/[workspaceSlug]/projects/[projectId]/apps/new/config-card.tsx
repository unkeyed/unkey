"use client";

import { ProjectDataProvider } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import { EnvironmentSettingsProvider } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/environment-provider";
import { match } from "@unkey/match";
import { type ReactNode, useState } from "react";
import { CardFrame } from "./card";
import { useNewAppFlow } from "./flow";
import { ConfigurePane } from "./panes/configure-pane";
import { ConfigureSkeleton } from "./panes/configure-skeleton";
import { ImagePane } from "./panes/image-pane";
import { PaneFooterProvider } from "./panes/pane-actions";
import { PickRepoPane } from "./panes/repository-pane";
import { SourcePane } from "./panes/source-pane";
import { type Card, type CardView, cardCopy, footerlessPanes } from "./wizard-model";

function WithApp({ appId, children }: { appId: string | null; children: ReactNode }) {
  const { projectId } = useNewAppFlow();
  return appId ? (
    <ProjectDataProvider projectId={projectId} appId={appId}>
      {children}
    </ProjectDataProvider>
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
  const [footer, setFooter] = useState<HTMLDivElement | null>(null);
  return (
    <PaneFooterProvider value={footer}>
      <CardFrame
        copy={cardCopy[card.pane]}
        eyebrow={label}
        back={back}
        actionsRef={setFooter}
        footerHidden={footerlessPanes.has(card.pane)}
      >
        <CardBody card={card} />
      </CardFrame>
    </PaneFooterProvider>
  );
}
