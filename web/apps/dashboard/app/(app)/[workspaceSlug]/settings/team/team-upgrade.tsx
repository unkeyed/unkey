"use client";

import type { DeployPlan } from "@/lib/stripe/deployPlan";
import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
} from "@unkey/ui";
import { useState } from "react";
import { PlansScreen } from "../billing/components/plans-screen";

export function TeamUpgrade({ currentPlan }: { currentPlan: DeployPlan | null }) {
  const [plansOpen, setPlansOpen] = useState(true);

  const description =
    currentPlan === "starter"
      ? "Starter doesn't include team members. Upgrade to Pro or Business, or add any API plan."
      : "Team members come with the Pro and Business Compute plans, and with every API plan.";

  return (
    <>
      <EmptyState className="w-full">
        <EmptyStateHeader className="gap-1">
          <EmptyStateTitle>Invite your team</EmptyStateTitle>
          <EmptyStateDescription>{description}</EmptyStateDescription>
        </EmptyStateHeader>
        <EmptyStateActions>
          <Button variant="primary" onClick={() => setPlansOpen(true)}>
            See plans
          </Button>
        </EmptyStateActions>
      </EmptyState>
      <PlansScreen
        open={plansOpen}
        onOpenChange={setPlansOpen}
        title="Invite your team"
        description={description}
        recommendedPlan="pro"
      />
    </>
  );
}
