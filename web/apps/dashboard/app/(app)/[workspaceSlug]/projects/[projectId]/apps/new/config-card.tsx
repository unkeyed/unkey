"use client";

import { match } from "@unkey/match";
import { cn } from "@unkey/ui/src/lib/utils";
import { type ReactNode, useState } from "react";
import { CardTitle, cardFooter } from "./card";
import { DeployPane } from "./panes/deploy-pane";
import { ImagePane } from "./panes/image-pane";
import { PaneFooterProvider } from "./panes/pane-actions";
import { ConfigureRepoPane, PickRepoPane } from "./panes/repository-pane";
import { SourcePane } from "./panes/source-pane";
import { VariablesPane } from "./panes/variables-pane";
import { scrollFadeClass, useScrollFade } from "./use-scroll-fade";
import { type ConfigCard as ConfigCardState, cardCopy, cardHasFooter } from "./wizard-model";

function CardBody({ card }: { card: ConfigCardState }) {
  return match(card)
    .with({ id: "source" }, ({ selected, lockedTo }) => (
      <SourcePane selected={selected} lockedTo={lockedTo} />
    ))
    .with({ id: "pick-repo" }, ({ appId }) => <PickRepoPane appId={appId} />)
    .with({ id: "configure-repo" }, ({ appId, focus }) => (
      <ConfigureRepoPane appId={appId} focus={focus} />
    ))
    .with({ id: "image" }, ({ appId }) => <ImagePane appId={appId} />)
    .with({ id: "variables" }, ({ appId }) => <VariablesPane appId={appId} />)
    .with({ id: "review" }, ({ appId, source }) => <DeployPane appId={appId} source={source} />)
    .exhaustive();
}

export function ConfigCard({ card, back }: { card: ConfigCardState; back: ReactNode }) {
  const [footer, setFooter] = useState<HTMLDivElement | null>(null);
  const copy = cardCopy[card.id];
  const scrollRef = useScrollFade();
  return (
    <PaneFooterProvider value={footer}>
      <div className="@container flex min-h-0 flex-col overflow-hidden rounded-lg border bg-raised">
        <div
          ref={scrollRef}
          className={cn(
            "flex min-h-0 flex-col gap-5 overflow-y-auto overscroll-contain p-5 [scrollbar-width:thin]",
            scrollFadeClass,
          )}
        >
          <div className="shrink-0">
            <CardTitle title={copy.title} description={copy.description} />
          </div>
          <CardBody card={card} />
        </div>
        <div className={cn(cardFooter, !cardHasFooter[card.id] && "hidden")}>
          {back}
          <div ref={setFooter} className="ml-auto flex items-center gap-3" />
        </div>
      </div>
    </PaneFooterProvider>
  );
}
