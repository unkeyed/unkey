"use client";

import { Button, cn } from "@unkey/ui";
import { type ReactNode, useState } from "react";
import { CardFrame } from "./card";
import { PaneFooterProvider } from "./panes/pane-actions";
import { type Pane, cardCopy, footerlessPanes } from "./wizard-model";

export type Role = "previous" | "current" | "next";

const COLUMN_GAP = "clamp(16px, 3vh, 48px)";
const COLUMN_TOP_SPACE = "clamp(24px, 5vh, 120px)";
const MIN_BOTTOM_PEEK_PX = 24;
export const COLUMN_CARD_MAX_HEIGHT = `min(880px, calc(100cqh - ${COLUMN_TOP_SPACE} - 2 * ${COLUMN_GAP} - ${MIN_BOTTOM_PEEK_PX}px))`;

export const roleCell: Record<Role, string> = {
  previous: "row-start-1 self-end",
  current: "row-start-2",
  next: "row-start-3 self-start",
};

export function FlowViewport({ children }: { children: ReactNode }) {
  return (
    <div className="absolute inset-0 flex flex-col bg-background [container-type:size]">
      {children}
    </div>
  );
}

export function FlowSplit({ steps, children }: { steps: ReactNode; children: ReactNode }) {
  return (
    <div className="flex min-h-0 flex-1 flex-col lg:flex-row">
      {steps}
      <div className="flex min-h-0 min-w-0 flex-1 flex-col">{children}</div>
    </div>
  );
}

export function StepAside({ children }: { children?: ReactNode }) {
  return (
    <aside
      aria-label="Steps"
      className="hidden w-18 shrink-0 flex-col justify-center border-r border-grayA-4 bg-gray-2 lg:flex xl:w-[max(25%,240px)] xl:px-5"
    >
      {children}
    </aside>
  );
}

export function ColumnGrid({ children }: { children: ReactNode }) {
  return (
    <div
      className="grid h-full grid-cols-[minmax(0,560px)] justify-center px-8"
      style={{
        rowGap: COLUMN_GAP,
        gridTemplateRows: `minmax(${COLUMN_TOP_SPACE}, 2fr) auto minmax(0, 3fr)`,
      }}
    >
      {children}
    </div>
  );
}

export function PendingFlowCard({ card, children }: { card: Pane; children: ReactNode }) {
  const [footer, setFooter] = useState<HTMLDivElement | null>(null);
  return (
    <FlowViewport>
      <FlowSplit steps={<StepAside />}>
        <div className="relative min-h-0 flex-1 overflow-hidden">
          <ColumnGrid>
            <div
              className={cn("relative col-start-1 flex min-h-0 flex-col", roleCell.current)}
              style={{ maxHeight: COLUMN_CARD_MAX_HEIGHT }}
            >
              <PaneFooterProvider value={footer}>
                <CardFrame
                  copy={cardCopy[card]}
                  actionsRef={setFooter}
                  footerHidden={footerlessPanes.has(card)}
                  back={
                    <Button variant="outline" size="sm" disabled>
                      Back
                    </Button>
                  }
                >
                  {children}
                </CardFrame>
              </PaneFooterProvider>
            </div>
          </ColumnGrid>
        </div>
      </FlowSplit>
    </FlowViewport>
  );
}
