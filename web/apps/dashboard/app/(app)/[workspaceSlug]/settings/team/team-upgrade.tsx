"use client";

import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
} from "@unkey/ui";
import { useState } from "react";
import { paywallCopy } from "../billing/components/paywall-copy";
import { PlansScreen } from "../billing/components/plans-screen";

export function TeamUpgrade() {
  const [plansOpen, setPlansOpen] = useState(true);
  const copy = paywallCopy("team");

  return (
    <>
      <EmptyState className="w-full">
        <EmptyStateHeader className="gap-1">
          <EmptyStateTitle>{copy.title}</EmptyStateTitle>
          <EmptyStateDescription>{copy.description}</EmptyStateDescription>
        </EmptyStateHeader>
        <EmptyStateActions>
          <Button variant="primary" onClick={() => setPlansOpen(true)}>
            See plans
          </Button>
        </EmptyStateActions>
      </EmptyState>
      <PlansScreen open={plansOpen} onOpenChange={setPlansOpen} reason="team" />
    </>
  );
}
