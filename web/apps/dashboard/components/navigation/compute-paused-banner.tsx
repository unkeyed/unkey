"use client";

import { pausedBody } from "@/app/(app)/[workspaceSlug]/settings/billing/components/compute-paused";
import { formatDollars } from "@/lib/fmt";
import { routes } from "@/lib/navigation/routes";
import { useWorkspace } from "@/providers/workspace-provider";
import Link from "next/link";
import { AppBanner } from "./app-banner";

/**
 * Workspace-wide banner for the spend-cap paused state. The cap is per
 * workspace and paused means compute is offline everywhere, so this sits at the
 * top of every page (mounted in the app layout) rather than only in billing.
 * Non-dismissible on purpose: it's a rare state that needs action.
 */
export function ComputePausedBanner() {
  const { workspace } = useWorkspace();

  if (!workspace?.deploySpendSuspended) {
    return null;
  }

  const budgetLabel =
    workspace.deploySpendBudgetCents != null
      ? formatDollars(workspace.deploySpendBudgetCents)
      : undefined;

  return (
    <AppBanner
      tone="warning"
      action={
        <Link href={routes.settings.billing({ workspaceSlug: workspace.slug })}>
          Manage billing
        </Link>
      }
    >
      <span className="font-semibold">Compute paused.</span> {pausedBody(budgetLabel)}
    </AppBanner>
  );
}
