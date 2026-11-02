"use client";

import { CardFrame } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/_components/flow/card";
import {
  COLUMN_CARD_MAX_HEIGHT,
  ColumnGrid,
  FlowSplit,
  FlowViewport,
  StepAside,
  roleCell,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/_components/flow/flow-layout";
import { Button, cn } from "@unkey/ui";
import type { ReactNode } from "react";
import { type Pane, cardCopy, footerlessPanes } from "./wizard-model";

export function PendingFlowCard({ card, children }: { card: Pane; children: ReactNode }) {
  return (
    <FlowViewport>
      <FlowSplit steps={<StepAside />}>
        <div className="relative min-h-0 flex-1 overflow-hidden">
          <ColumnGrid>
            <div
              className={cn("relative col-start-1 flex min-h-0 flex-col", roleCell.current)}
              style={{ maxHeight: COLUMN_CARD_MAX_HEIGHT }}
            >
              <CardFrame
                copy={cardCopy[card]}
                footerHidden={footerlessPanes.has(card)}
                back={
                  <Button variant="outline" size="sm" disabled>
                    Back
                  </Button>
                }
              >
                {children}
              </CardFrame>
            </div>
          </ColumnGrid>
        </div>
      </FlowSplit>
    </FlowViewport>
  );
}
