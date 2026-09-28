"use client";

import { IconUserPlusOutline18 } from "@unkey/icons";
import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
} from "@unkey/ui";
import { useState } from "react";
import { paywallCopy } from "../billing/components/paywall-copy";
import { PlansScreen } from "../billing/components/plans-screen";

export function TeamUpgrade() {
  const [plansOpen, setPlansOpen] = useState(false);
  const copy = paywallCopy("team");

  return (
    <>
      <EmptyState className="w-full">
        <EmptyStateIcon>
          <IconUserPlusOutline18 />
        </EmptyStateIcon>
        <EmptyStateHeader className="gap-1">
          <EmptyStateTitle>{copy.title}</EmptyStateTitle>
          <EmptyStateDescription>{copy.description}</EmptyStateDescription>
        </EmptyStateHeader>
        <EmptyStateActions>
          <Button variant="primary" onClick={() => setPlansOpen(true)}>
            Upgrade
          </Button>
        </EmptyStateActions>
      </EmptyState>
      <PlansScreen open={plansOpen} onOpenChange={setPlansOpen} reason="team" />
    </>
  );
}
